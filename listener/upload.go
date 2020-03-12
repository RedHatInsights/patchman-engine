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
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"time"
)

const (
	WarnSkippingNoPackages = "skipping profile with no packages"
	ErrorNoAccountProvided = "no account provided in host message"
	ErrorKafkaSend         = "unable to send evaluation message"
	ErrorProcessUpload     = "unable to process upload"
	UploadSuccessNoEval    = "upload event handled successfully, no eval required"
	UploadSuccess          = "upload event handled successfully"
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
		utils.Log("inventoryID", event.Host.ID).Warn(WarnSkippingNoPackages)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedWarnNoPackages).Inc()
		return
	}

	if len(event.Host.Account) == 0 {
		utils.Log("inventoryID", event.Host.ID).Error(ErrorNoAccountProvided)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorIdentity).Inc()
		return
	}

	sys, err := processUpload(event.Host.Account, &event.Host)
	if err != nil {
		utils.Log("inventoryID", event.Host.ID, "err", err.Error()).Error(ErrorProcessUpload)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorProcessing).Inc()
		return
	}

	if sys.UnchangedSince != nil && sys.LastEvaluation != nil {
		if sys.UnchangedSince.Before(*sys.LastEvaluation) {
			messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedSuccessNoEval).Inc()
			utils.Log("inventoryID", event.Host.ID).Debug(UploadSuccessNoEval)
			return
		}
	}

	err = mqueue.WriteEvents(context.Background(), evalWriter, mqueue.PlatformEvent{ID: sys.InventoryID})
	if err != nil {
		utils.Log("inventoryID", event.Host.ID, "err", err.Error()).Error(ErrorKafkaSend)
		return
	}

	messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedSuccess).Inc()
	utils.Log("inventoryID", event.Host.ID).Debug(UploadSuccess)
}

// Stores or updates the account data, returning the account id
func getOrCreateAccount(tx *gorm.DB, account string) (int, error) {
	rhAccount := models.RhAccount{
		Name: account,
	}
	// Select, and only if not found attempt an insertion
	tx.Where("name = ?", account).Find(&rhAccount)
	if rhAccount.ID != 0 {
		return rhAccount.ID, nil
	}
	err := database.OnConflictUpdate(tx, "name", "name").Create(&rhAccount).Error
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

// nolint: funlen
// Stores or updates base system profile, returing internal system id
func updateSystemPlatform(tx *gorm.DB, inventoryID string, accountID int, host *Host,
	updatesReq *vmaas.UpdatesV3Request) (*models.SystemPlatform, error) {
	updatesReqJSON, err := json.Marshal(updatesReq)
	if err != nil {
		return nil, errors.Wrap(err, "Serializing vmaas request")
	}

	hash := sha256.New()
	// Never returns an error
	hash.Write(updatesReqJSON) // nolint: errcheck

	jsonChecksum := hex.EncodeToString(hash.Sum([]byte{}))
	var colsToUpdate = []string{
		"last_upload",
		"stale",
		"stale_timestamp",
		"stale_warning_timestamp",
		"culled_timestamp"}

	now := time.Now()

	systemPlatform := models.SystemPlatform{
		InventoryID:           inventoryID,
		RhAccountID:           accountID,
		VmaasJSON:             string(updatesReqJSON),
		JSONChecksum:          jsonChecksum,
		LastUpload:            &now,
		Stale:                 false,
		StaleTimestamp:        optParseTimestamp(host.StaleTimestamp),
		StaleWarningTimestamp: optParseTimestamp(host.StaleWarningTimestamp),
		CulledTimestamp:       optParseTimestamp(host.CulledTimestamp),
	}

	var sameJSONCount int
	// Will return sameJSONCount if json_checksum HAVEN'T CHANGED
	tx.Table("system_platform").
		Where("inventory_id = ?", inventoryID).
		Where("json_checksum = ?", jsonChecksum).
		Count(&sameJSONCount)

	// Skip updating vmaas_json if the checksum haven't changed. Should reduce TOAST trashing
	if sameJSONCount == 0 {
		colsToUpdate = append(colsToUpdate, "vmaas_json", "json_checksum")
	}

	query := database.OnConflictUpdate(tx, "inventory_id", colsToUpdate...).
		Save(&systemPlatform)

	if query.Error != nil {
		return nil, errors.Wrap(query.Error, "Unable to save or update system in database")
	}

	addedRepos, addedSysRepos, deletedSysRepos, err := updateRepos(tx, systemPlatform.ID, updatesReq.RepositoryList)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update system repos")
	}

	utils.Log("inventoryID", inventoryID, "packages", len(updatesReq.PackageList), "repos",
		len(updatesReq.RepositoryList), "modules", len(updatesReq.ModulesList),
		"addedRepos", addedRepos, "addedSysRepos", addedSysRepos, "deletedSysRepos", deletedSysRepos).
		Debug("System created or updated successfully")
	return &systemPlatform, nil
}

func updateRepos(tx *gorm.DB, systemID int, repos []string) (addedRepos int64, addedSysRepos int64,
	deletedSysRepos int, err error) {
	repoIDs, addedRepos, err := ensureReposInDb(tx, repos)
	if err != nil {
		return 0, 0, 0, err
	}

	addedSysRepos, deletedSysRepos, err = updateSystemRepos(tx, systemID, repoIDs)
	if err != nil {
		return 0, 0, 0, err
	}
	return addedRepos, addedSysRepos, deletedSysRepos, nil
}

func ensureReposInDb(tx *gorm.DB, repos []string) (repoIDs []int, added int64, err error) {
	repoObjs := make(models.RepoSlice, len(repos))
	for i, repo := range repos {
		repoObjs[i] = models.Repo{Name: repo}
	}

	txOnConflict := tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	err = database.BulkInsert(txOnConflict, repoObjs)
	if err != nil {
		return nil, 0, errors.Wrap(err, "unable to update repos")
	}
	added = txOnConflict.RowsAffected
	reposAddedCnt.Add(float64(added))

	err = tx.Model(&models.Repo{}).Where("name IN (?)", repos).
		Pluck("id", &repoIDs).Error
	if err != nil {
		return nil, 0, errors.Wrap(err, "unable to load repos IDs")
	}

	return repoIDs, added, nil
}

func updateSystemRepos(tx *gorm.DB, systemID int, repoIDs []int) (nAdded int64, nDeleted int, err error) {
	repoSystemObjs := make(models.SystemRepoSlice, len(repoIDs))
	for i, repoID := range repoIDs {
		repoSystemObjs[i] = models.SystemRepo{SystemID: systemID, RepoID: repoID}
	}

	txOnConflict := tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	err = database.BulkInsert(txOnConflict, repoSystemObjs)
	if err != nil {
		return 0, 0, errors.Wrap(err, "unable to update system repos")
	}
	nAdded = txOnConflict.RowsAffected

	nDeleted, err = deleteOtherSystemRepos(tx, systemID, repoIDs)
	if err != nil {
		return nAdded, 0, errors.Wrap(err, "unable to delete out-of-date system repos")
	}

	return nAdded, nDeleted, nil
}

func deleteOtherSystemRepos(tx *gorm.DB, systemID int, repoIDs []int) (nDeleted int, err error) {
	type result struct{ DeletedCount int }
	var res result
	if len(repoIDs) > 0 {
		err = tx.Raw("WITH deleted AS "+ // to count deleted items
			"(DELETE FROM system_repo WHERE system_id = ? AND repo_id NOT IN (?) RETURNING repo_id) "+
			"SELECT count(*) AS deleted_count FROM deleted", systemID, repoIDs).Scan(&res).Error
	} else {
		err = tx.Raw("WITH deleted AS "+
			"(DELETE FROM system_repo WHERE system_id = ? RETURNING repo_id) "+
			"SELECT count(*) AS deleted_count FROM deleted", systemID).Scan(&res).Error
	}
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

// nolint: funlen
// We have received new upload, update stored host data, and re-evaluate the host against VMaaS
func processUpload(account string, host *Host) (*models.SystemPlatform, error) {
	tx := database.Db.Begin()
	// Ensure we have account stored
	accountID, err := getOrCreateAccount(tx, account)
	if err != nil {
		return nil, errors.Wrap(err, "saving account into the database")
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

	sys, err := updateSystemPlatform(tx, host.ID, accountID, host, &updatesReq)
	if err != nil {
		tx.Rollback()
		return nil, errors.Wrap(err, "saving system into the database")
	}
	err = tx.Commit().Error
	if err != nil {
		return nil, errors.Wrap(err, "saving system into the database")
	}
	return sys, nil
}
