package listener

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"strings"
	"time"
)

const (
	WarnSkippingNoPackages = "skipping profile with no packages"
	WarnSkippingReporter   = "skipping excluded reporter"
	ErrorNoAccountProvided = "no account provided in host message"
	ErrorKafkaSend         = "unable to send evaluation message"
	ErrorProcessUpload     = "unable to process upload"
	UploadSuccessNoEval    = "upload event handled successfully, no eval required"
	UploadSuccess          = "upload event handled successfully"
)

var DeletionThreshold = time.Hour * time.Duration(utils.GetIntEnvOrDefault("SYSTEM_DELETE_HRS", 4))

type Host struct {
	ID                    string                                       `json:"id,omitempty"`
	DisplayName           *string                                      `json:"display_name,omitempty"`
	AnsibleHost           *string                                      `json:"ansible_host,omitempty"`
	Account               string                                       `json:"account,omitempty"`
	InsightsID            string                                       `json:"insights_id,omitempty"`
	RhelMachineID         string                                       `json:"rhel_machine_id,omitempty"`
	SubscriptionManagerID string                                       `json:"subscription_manager_id,omitempty"`
	SatelliteID           string                                       `json:"satellite_id,omitempty"`
	FQDN                  string                                       `json:"fqdn,omitempty"`
	BiosUUID              string                                       `json:"bios_uuid,omitempty"`
	IPAddresses           []string                                     `json:"ip_addresses,omitempty"`
	MacAddresses          []string                                     `json:"mac_addresses,omitempty"`
	ExternalID            string                                       `json:"external_id,omitempty"`
	Created               string                                       `json:"created,omitempty"`
	Updated               string                                       `json:"updated,omitempty"`
	StaleTimestamp        *base.Rfc3339Timestamp                       `json:"stale_timestamp,omitempty"`
	StaleWarningTimestamp *base.Rfc3339Timestamp                       `json:"stale_warning_timestamp,omitempty"`
	CulledTimestamp       *base.Rfc3339Timestamp                       `json:"culled_timestamp,omitempty"`
	Reporter              string                                       `json:"reporter,omitempty"`
	Tags                  []inventory.StructuredTag                    `json:"tags,omitempty"`
	SystemProfile         inventory.SystemProfileSpecYamlSystemProfile `json:"system_profile,omitempty"`
}

type HostEvent struct {
	Type             string                 `json:"type"`
	PlatformMetadata map[string]interface{} `json:"platform_metadata"`
	Host             Host                   `json:"host"`
}

func HandleUpload(event HostEvent) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageHandlingDuration.WithLabelValues(EventUpload))

	updateReporterCounter(event.Host.Reporter)

	if _, ok := excludedReporters[event.Host.Reporter]; ok {
		utils.Log("inventoryID", event.Host.ID, "reporter", event.Host.Reporter).Warn(WarnSkippingReporter)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedWarnExcludedReporter).Inc()
		return nil
	}

	if len(event.Host.Account) == 0 {
		utils.Log("inventoryID", event.Host.ID).Error(ErrorNoAccountProvided)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorIdentity).Inc()
		return nil
	}

	if len(event.Host.SystemProfile.InstalledPackages) == 0 {
		utils.Log("inventoryID", event.Host.ID).Warn(WarnSkippingNoPackages)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedWarnNoPackages).Inc()
		return nil
	}

	sys, err := processUpload(event.Host.Account, &event.Host)

	if err != nil {
		utils.Log("inventoryID", event.Host.ID, "err", err.Error()).Error(ErrorProcessUpload)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorProcessing).Inc()
		return errors.Wrap(err, "Could not process upload")
	}

	// Deleted system, return nil
	if sys == nil {
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedDeleted).Inc()
		return nil
	}

	if sys.UnchangedSince != nil && sys.LastEvaluation != nil {
		if sys.UnchangedSince.Before(*sys.LastEvaluation) {
			messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedSuccessNoEval).Inc()
			utils.Log("inventoryID", event.Host.ID).Debug(UploadSuccessNoEval)
			return nil
		}
	}
	t := base.Rfc3339Timestamp(time.Now())
	ev := mqueue.PlatformEvent{ID: sys.InventoryID, AccountID: sys.RhAccountID, Timestamp: &t}
	// Not sending evaluation message is a fatal error
	err = mqueue.WriteEvents(base.Context, evalWriter, ev)
	if err != nil {
		utils.Log("inventoryID", event.Host.ID, "err", err.Error()).Error(ErrorKafkaSend)
		return errors.Wrap(err, "Could send eval message")
	}

	messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedSuccess).Inc()
	utils.Log("inventoryID", event.Host.ID).Debug(UploadSuccess)
	return nil
}

func updateReporterCounter(reporter string) {
	if _, ok := validReporters[reporter]; ok {
		receivedFromReporter.WithLabelValues(reporter).Inc()
	} else {
		receivedFromReporter.WithLabelValues("unknown").Inc()
		utils.Log("reporter", reporter).Warn("unknown reporter")
	}
}

// Stores or updates the account data, returning the account id
func getOrCreateAccount(account string) (int, error) {
	rhAccount := models.RhAccount{
		Name: account,
	}
	// Select, and only if not found attempt an insertion
	database.Db.Where("name = ?", account).Find(&rhAccount)
	if rhAccount.ID != 0 {
		return rhAccount.ID, nil
	}
	err := database.OnConflictUpdate(database.Db, "name", "name").Create(&rhAccount).Error
	return rhAccount.ID, err
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
		"display_name",
		"last_upload",
		"stale",
		"stale_timestamp",
		"stale_warning_timestamp",
		"culled_timestamp",
	}

	now := time.Now()
	displayName := inventoryID
	if host.DisplayName != nil && len(*host.DisplayName) > 0 {
		displayName = *host.DisplayName
	}

	staleWarning := host.StaleWarningTimestamp.Time()

	systemPlatform := models.SystemPlatform{
		InventoryID:           inventoryID,
		RhAccountID:           accountID,
		DisplayName:           displayName,
		VmaasJSON:             string(updatesReqJSON),
		JSONChecksum:          jsonChecksum,
		LastUpload:            &now,
		StaleTimestamp:        host.StaleTimestamp.Time(),
		StaleWarningTimestamp: host.StaleWarningTimestamp.Time(),
		CulledTimestamp:       host.CulledTimestamp.Time(),
		Stale:                 staleWarning != nil && staleWarning.Before(time.Now()),
		ReporterID:            getReporterID(host.Reporter),
	}

	var oldJSONChecksum []string
	// Lock the row for update & return checksum
	tx.Set("gorm:query_option", "FOR UPDATE OF system_platform").
		Table("system_platform").
		Where("inventory_id = ?::uuid", inventoryID).
		Where("rh_account_id = ?", accountID).
		Pluck("json_checksum", &oldJSONChecksum)

	shouldUpdateRepos := false
	var addedRepos, addedSysRepos, deletedSysRepos int64

	// Skip updating vmaas_json if the checksum haven't changed. Should reduce TOAST trashing
	if len(oldJSONChecksum) == 0 || oldJSONChecksum[0] != jsonChecksum {
		colsToUpdate = append(colsToUpdate, "vmaas_json", "json_checksum", "reporter_id")
		shouldUpdateRepos = true
	}

	if err := database.OnConflictUpdateMulti(tx, []string{"rh_account_id", "inventory_id"}, colsToUpdate...).
		Save(&systemPlatform).Error; err != nil {
		return nil, errors.Wrap(err, "Unable to save or update system in database")
	}

	if shouldUpdateRepos {
		// We also don't need to update repos if vmaas_json haven't changed
		addedRepos, addedSysRepos, deletedSysRepos, err = updateRepos(tx, accountID,
			systemPlatform.ID, updatesReq.RepositoryList)
		if err != nil {
			utils.Log("repository_list", updatesReq.RepositoryList, "inventoryID", systemPlatform.ID).
				Error("repos failed to insert")
			return nil, errors.Wrap(err, "unable to update system repos")
		}
	}

	utils.Log("inventoryID", inventoryID, "packages", len(updatesReq.PackageList), "repos",
		len(updatesReq.RepositoryList), "modules", len(updatesReq.ModulesList),
		"addedRepos", addedRepos, "addedSysRepos", addedSysRepos, "deletedSysRepos", deletedSysRepos).
		Debug("System created or updated successfully")
	return &systemPlatform, nil
}

func getReporterID(reporter string) *int {
	if id, ok := validReporters[reporter]; ok {
		return &id
	}
	utils.Log("reporter", reporter).Warn("no reporter id found, returning nil")
	return nil
}

func updateRepos(tx *gorm.DB, rhAccountID int, systemID int, repos []string) (addedRepos int64, addedSysRepos int64,
	deletedSysRepos int64, err error) {
	repoIDs, addedRepos, err := ensureReposInDB(tx, repos)
	if err != nil {
		return 0, 0, 0, err
	}

	addedSysRepos, deletedSysRepos, err = updateSystemRepos(tx, rhAccountID, systemID, repoIDs)
	if err != nil {
		return 0, 0, 0, err
	}
	return addedRepos, addedSysRepos, deletedSysRepos, nil
}

func ensureReposInDB(tx *gorm.DB, repos []string) (repoIDs []int, added int64, err error) {
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

func updateSystemRepos(tx *gorm.DB, rhAccountID int, systemID int, repoIDs []int) (
	nAdded int64, nDeleted int64, err error) {
	repoSystemObjs := make(models.SystemRepoSlice, len(repoIDs))
	for i, repoID := range repoIDs {
		repoSystemObjs[i] = models.SystemRepo{RhAccountID: rhAccountID, SystemID: systemID, RepoID: repoID}
	}

	txOnConflict := tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	err = database.BulkInsert(txOnConflict, repoSystemObjs)
	if err != nil {
		return 0, 0, errors.Wrap(err, "unable to update system repos")
	}
	nAdded = txOnConflict.RowsAffected

	nDeleted, err = deleteOtherSystemRepos(tx, rhAccountID, systemID, repoIDs)
	if err != nil {
		return nAdded, 0, errors.Wrap(err, "unable to delete out-of-date system repos")
	}

	return nAdded, nDeleted, nil
}

func deleteOtherSystemRepos(tx *gorm.DB, rhAccountID int, systemID int, repoIDs []int) (nDeleted int64, err error) {
	type result struct{ DeletedCount int64 }
	var res result
	if len(repoIDs) > 0 {
		err = tx.Raw("WITH deleted AS "+ // to count deleted items
			"(DELETE FROM system_repo WHERE rh_account_id = ? AND system_id = ? AND repo_id NOT IN (?) RETURNING repo_id) "+
			"SELECT count(*) AS deleted_count FROM deleted", rhAccountID, systemID, repoIDs).Scan(&res).Error
	} else {
		err = tx.Raw("WITH deleted AS "+
			"(DELETE FROM system_repo WHERE rh_account_id = ? AND system_id = ? RETURNING repo_id) "+
			"SELECT count(*) AS deleted_count FROM deleted", rhAccountID, systemID).Scan(&res).Error
	}
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

func processRepos(systemProfile *inventory.SystemProfileSpecYamlSystemProfile) []string {
	repos := make([]string, 0, len(systemProfile.YumRepos))
	for _, r := range systemProfile.YumRepos {
		if len(strings.TrimSpace(r.Id)) == 0 {
			utils.Log("repo", r.Id).Warn("removed repo with invalid name")
			continue
		}

		if r.Enabled {
			repos = append(repos, r.Id)
		}
	}
	return repos
}

func processModules(systemProfile *inventory.SystemProfileSpecYamlSystemProfile) []vmaas.UpdatesRequestModulesList {
	var modules []vmaas.UpdatesRequestModulesList
	if count := len(systemProfile.DnfModules); count > 0 {
		modules = make([]vmaas.UpdatesRequestModulesList, count)
		for i, m := range systemProfile.DnfModules {
			modules[i] = vmaas.UpdatesRequestModulesList{
				ModuleName:   m.Name,
				ModuleStream: m.Stream,
			}
		}
	}
	return modules
}

// We have received new upload, update stored host data, and re-evaluate the host against VMaaS
func processUpload(account string, host *Host) (*models.SystemPlatform, error) {
	// Ensure we have account stored
	accountID, err := getOrCreateAccount(account)
	if err != nil {
		return nil, errors.Wrap(err, "saving account into the database")
	}

	systemProfile := host.SystemProfile
	// Prepare VMaaS request
	updatesReq := vmaas.UpdatesV3Request{
		PackageList:    systemProfile.InstalledPackages,
		RepositoryList: processRepos(&systemProfile),
		ModulesList:    processModules(&systemProfile),
		Basearch:       systemProfile.Arch,
		SecurityOnly:   false,
		LatestOnly:     true,
	}

	// use rhsm version if set
	releasever := systemProfile.Rhsm.Version
	if len(releasever) > 0 {
		updatesReq.Releasever = releasever
	}

	tx := database.Db.BeginTx(base.Context, nil)
	defer tx.RollbackUnlessCommitted()

	var deleted models.DeletedSystem
	if err := tx.Find(&deleted, "inventory_id = ?", host.ID).Error; err != nil &&
		!gorm.IsRecordNotFoundError(err) {
		return nil, errors.Wrap(err, "Checking deleted systems")
	}

	// If the system was deleted in last hour, don't register this upload
	if deleted.InventoryID != "" && deleted.WhenDeleted.After(time.Now().Add(-DeletionThreshold)) {
		utils.Log("inventoryID", host.ID).Info("Received recently deleted system")
		return nil, nil
	}
	sys, err := updateSystemPlatform(tx, host.ID, accountID, host, &updatesReq)
	if err != nil {
		return nil, errors.Wrap(err, "saving system into the database")
	}
	err = tx.Commit().Error
	if err != nil {
		return nil, errors.Wrap(err, "Committing changes")
	}
	return sys, nil
}
