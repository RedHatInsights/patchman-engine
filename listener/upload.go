package listener

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"time"
)

type Host struct {
	ID                    string                    `json:"id,omitempty"`
	DisplayName           *string                   `json:"display_name,omitempty"`
	AnsibleHost           *string                   `json:"ansible_host,omitempty"`
	Account               string                    `json:"account,omitempty"`
	InsightsID            string                    `json:"insights_id,omitempty"`
	RhelMachineID         string                    `json:"rhel_machine_id,omitempty"`
	SubscriptionManagerID string                    `json:"subscription_manager_id,omitempty"`
	SatelliteID           string                    `json:"satellite_id,omitempty"`
	FQDN                  string                    `json:"fqdn,omitempty"`
	BiosUUID              string                    `json:"bios_uuid,omitempty"`
	IPAddresses           []string                  `json:"ip_addresses,omitempty"`
	MacAddresses          []string                  `json:"mac_addresses,omitempty"`
	ExternalID            string                    `json:"external_id,omitempty"`
	Created               string                    `json:"created,omitempty"`
	Updated               string                    `json:"updated,omitempty"`
	StaleTimestamp        *string                   `json:"stale_timestamp,omitempty"`
	StaleWarningTimestamp *string                   `json:"stale_warning_timestamp,omitempty"`
	CulledTimestamp       *string                   `json:"culled_timestamp,omitempty"`
	Reporter              string                    `json:"reporter,omitempty"`
	Tags                  []inventory.StructuredTag `json:"tags,omitempty"`
	SystemProfile         inventory.SystemProfileIn `json:"system_profile,omitempty"`
}

type HostEgressEvent struct {
	Type             string                 `json:"type"`
	PlatformMetadata map[string]interface{} `json:"platform_metadata"`
	Host             Host                   `json:"host"`
}

func uploadMsgHandler(msg kafka.Message) {
	var event HostEgressEvent
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to parse upload msg")
		utils.Log("raw", string(msg.Value)).Trace("Raw message string")
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorParsing).Inc()
		// TODO: Forcing a panic here to quickly discover whether we have correct message format
		panic(err)
	}
	uploadHandler(event)
}

func uploadHandler(event HostEgressEvent) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageHandlingDuration.WithLabelValues(EventUpload))

	systemProfile := event.Host.SystemProfile

	if len(systemProfile.InstalledPackages) == 0 {
		utils.Log("inventoryID", event.Host.ID).Warn("skipping profile with no packages")
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedWarnNoPackages).Inc()
		return
	}

	if len(event.Host.Account) == 0 {
		utils.Log("inventoryID", event.Host.ID).Error("No account provided in host message")
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorIdentity)
		return
	}

	err := processUpload(context.Background(), event.Host.ID, event.Host.Account, &event.Host)
	if err != nil {
		utils.Log("inventoryID", event.Host.ID, "err", err.Error()).Error("unable to process upload")
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorProcessing).Inc()
		return
	}

	messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedSuccess).Inc()
	utils.Log("inventoryID", event.Host.ID).Debug("Upload event handled successfully")
}

// Stores or updates the account data, returning the account id
func getOrCreateAccount(account string) (int, error) {
	rhAccount := models.RhAccount{
		Name: account,
	}

	err := database.OnConflictUpdate(database.Db, "name", "name").Create(&rhAccount).Error
	return rhAccount.ID, err
}

func optParseTimestamp(t *string) *time.Time {
	if t == nil || len(*t) == 0 {
		return nil
	}
	v, err := time.Parse(base.Rfc3339NoTz, *t)
	if err != nil {
		utils.Log("err", err.Error()).Error("Opt timestamp parse")
		return nil
	}
	return &v
}

// Stores or updates base system profile, returing internal system id
func updateSystemPlatform(inventoryID string, accountID int, host *Host,
	updatesReq *vmaas.UpdatesV3Request) (*models.SystemPlatform, error) {
	updatesReqJSON, err := json.Marshal(updatesReq)
	if err != nil {
		return nil, errors.Wrap(err, "Serializing vmaas request")
	}

	hash := sha256.New()
	_, err = hash.Write(updatesReqJSON)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to hash updates json")
	}

	jsonChecksum := hex.EncodeToString(hash.Sum([]byte{}))

	now := time.Now()

	systemPlatform := models.SystemPlatform{
		InventoryID:  inventoryID,
		RhAccountID:  accountID,
		VmaasJSON:    string(updatesReqJSON),
		JSONChecksum: jsonChecksum,
		LastUpload:   &now,

		StaleTimestamp:        optParseTimestamp(host.StaleTimestamp),
		StaleWarningTimestamp: optParseTimestamp(host.StaleWarningTimestamp),
		CulledTimestamp:       optParseTimestamp(host.CulledTimestamp),
	}

	tx := database.OnConflictUpdate(database.Db, "inventory_id", "vmaas_json", "json_checksum",
		"last_upload", "stale_timestamp", "stale_warning_timestamp", "culled_timestamp")
	retTx := tx.Create(&systemPlatform)
	if retTx.Error != nil {
		return nil, errors.Wrap(retTx.Error, "Unable to save or update system in database")
	}

	if retTx.RowsAffected == 0 {
		return nil, errors.New("System neither created nor updated")
	}

	utils.Log("inventoryID", inventoryID, "packages", len(updatesReq.PackageList), "repos",
		len(updatesReq.RepositoryList), "modules", len(updatesReq.ModulesList)).
		Debug("System created or updated successfully")

	return &systemPlatform, nil
}

// nolint: funlen
// We have received new upload, update stored host data, and re-evaluate the host against VMaaS
func processUpload(ctx context.Context, inventoryID string, account string, host *Host) error {
	// Ensure we have account stored
	accountID, err := getOrCreateAccount(account)
	if err != nil {
		return errors.Wrap(err, "saving account into the database")
	}

	systemProfile := host.SystemProfile
	// Prepare VMaaS request
	updatesReq := vmaas.UpdatesV3Request{
		PackageList:  systemProfile.InstalledPackages,
		Basearch:     systemProfile.Arch,
		SecurityOnly: false,
	}

	if count := len(systemProfile.DnfModules); count > 0 {
		updatesReq.ModulesList = make([]vmaas.UpdatesRequestModulesList, count)
		for i, m := range systemProfile.DnfModules {
			updatesReq.ModulesList[i] = vmaas.UpdatesRequestModulesList{
				ModuleName:   m.Name,
				ModuleStream: m.Stream,
			}
		}
	}

	updatesReq.RepositoryList = make([]string, len(systemProfile.YumRepos))
	for i, r := range systemProfile.YumRepos {
		if r.Enabled {
			updatesReq.RepositoryList[i] = r.Id
		}
	}

	_, err = updateSystemPlatform(host.ID, accountID, host, &updatesReq)
	if err != nil {
		return errors.Wrap(err, "saving system into the database")
	}

	event := mqueue.PlatformEvent{
		ID: inventoryID,
	}
	err = mqueue.WriteEvents(ctx, evalWriter, event)
	if err != nil {
		return errors.Wrap(err, "Sending kafka event failed")
	}
	return nil
}
