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
	"time"
)

func uploadHandler(event mqueue.PlatformEvent) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageHandlingDuration.WithLabelValues(EventUpload))

	if event.B64Identity == nil {
		utils.Log().Error("Identity not provided")
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorIdentity).Inc()
		return
	}

	identity, err := parseUploadMessage(&event)
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to parse upload msg")
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorParsing).Inc()
		return
	}

	err = processUpload(event.ID, identity.Identity.AccountNumber, *event.B64Identity)
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to process upload")
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorProcessing).Inc()
		return
	}

	messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedSuccess).Inc()
}

func parseUploadMessage(event *mqueue.PlatformEvent) (*utils.Identity, error) {
	// We need the b64 identity in order to call the inventory
	if event.B64Identity == nil {
		utils.Log().Error("No identity provided")
		return nil, errors.New("No identity provided")
	}

	identity, err := utils.ParseIdentity(*event.B64Identity)
	if err != nil {
		utils.Log("err", err.Error()).Error("Could not parse identity")
		return nil, errors.New("Could not parse identity")
	}

	if !identity.IsSmartEntitled() {
		utils.Log("account", identity.Identity.AccountNumber).Info("Is not smart entitled")
		return nil, errors.New("Is not smart entitled")
	}
	return identity, nil
}

// Stores or updates the account data, returning the account id
func getOrCreateAccount(account string) (int, error) {
	rhAccount := models.RhAccount{
		Name: account,
	}

	err := database.OnConflictUpdate(database.Db, "name", "name").Create(&rhAccount).Error
	return rhAccount.ID, err
}

func optParseTimestap(t *string) *time.Time {
	if t != nil && len(*t) > 0 {
		v, err := time.Parse(base.Rfc3339NoTz, *t)
		if err == nil {
			return &v
		}
	}
	return nil
}

// Stores or updates base system profile, returing internal system id
func updateSystemPlatform(inventoryID string, accountID int,
	invData *inventory.HostOut, updatesReq *vmaas.UpdatesV3Request) (*models.SystemPlatform, error) {
	updatesReqJSON, err := json.Marshal(&updatesReq)
	if err != nil {
		utils.Log("err", err.Error()).Error("Serializing vmaas request")
		return nil, err
	}

	hash := sha256.New()
	_, err = hash.Write(updatesReqJSON)
	if err != nil {
		utils.Log("err", err.Error()).Error("Unable to hash updates json")
		return nil, err
	}

	jsonChecksum := hex.EncodeToString(hash.Sum([]byte{}))

	now := time.Now()

	systemPlatform := models.SystemPlatform{
		InventoryID:    inventoryID,
		RhAccountID:    accountID,
		VmaasJSON:      string(updatesReqJSON),
		JSONChecksum:   jsonChecksum,
		LastEvaluation: nil,
		LastUpload:     &now,

		StaleTimestamp:        optParseTimestap(invData.StaleTimestamp),
		StaleWarningTimestamp: optParseTimestap(invData.StaleWarningTimestamp),
		CulledTimestamp:       optParseTimestap(invData.CulledTimestamp),
	}

	tx := database.OnConflictUpdate(database.Db, "inventory_id", "vmaas_json", "json_checksum",
		"last_evaluation", "last_upload")
	err = tx.Create(&systemPlatform).Error

	if err != nil {
		utils.Log("err", err.Error()).Error("Saving host into the database")
		return nil, err
	}
	return &systemPlatform, nil
}

// nolint: funlen
// We have received new upload, update stored host data, and re-evaluate the host against VMaaS
func processUpload(hostID string, account string, identity string) error {
	utils.Log("hostID", hostID).Debug("Downloading system profile")

	apiKey := inventory.APIKey{Prefix: "", Key: identity}
	// Create new context, which has the apikey value set. This is then used as a value for `x-rh-identity`
	ctx := context.WithValue(context.Background(), inventory.ContextAPIKey, apiKey)

	hostResults, _, err := inventoryClient.HostsApi.ApiHostGetHostById(ctx, []string{hostID}, nil)
	if err != nil {
		return errors.Wrap(err, "could not query inventory")
	}

	profileResults, _, err := inventoryClient.HostsApi.ApiHostGetHostSystemProfileById(ctx, []string{hostID}, nil)
	if err != nil {
		return errors.Wrap(err, "could not inventory system profile")
	}

	utils.Log().Debug("System profile download complete")

	if profileResults.Count == 0 || hostResults.Count == 0 {
		return errors.Wrap(err, "no system details returned, systemProfile is probably deleted")
	}

	// We only process one systemProfile per message here
	host := hostResults.Results[0]
	systemProfile := profileResults.Results[0]
	// Ensure we have account stored
	accountID, err := getOrCreateAccount(account)
	if err != nil {
		return errors.Wrap(err, "saving account into the database")
	}

	modules := make([]vmaas.UpdatesRequestModulesList, len(systemProfile.SystemProfile.DnfModules))
	for i, m := range systemProfile.SystemProfile.DnfModules {
		modules[i] = vmaas.UpdatesRequestModulesList{
			ModuleName:   m.Name,
			ModuleStream: m.Stream,
		}
	}
	repos := []string{}
	for _, r := range systemProfile.SystemProfile.YumRepos {
		repos = append(repos, r.Id)
	}
	// Prepare VMaaS request
	updatesReq := vmaas.UpdatesV3Request{
		PackageList:    systemProfile.SystemProfile.InstalledPackages,
		Basearch:       systemProfile.SystemProfile.Arch,
		ModulesList:    modules,
		RepositoryList: repos,
		SecurityOnly:   false,
	}

	_, err = updateSystemPlatform(systemProfile.Id, accountID, &host, &updatesReq)
	if err != nil {
		return errors.Wrap(err, "saving system into the database")
	}

	event := mqueue.PlatformEvent{
		ID: hostID,
	}

	utils.Log().Debug("Sending evaluation kafka message")
	err = evalWriter.WriteEvent(ctx, event)
	if err != nil {
		return errors.Wrap(err, "Sending kafka event failed")
	}
	return nil
}
