package listener

import (
	"app/base"
	"app/base/database"
	"app/base/inventory"
	"app/base/models"
	"app/base/mqueue"
	"app/base/types"
	"app/base/utils"
	"app/base/vmaas"
	"app/manager/middlewares"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	WarnSkippingNoPackages = "skipping profile with no packages"
	WarnSkippingReporter   = "skipping excluded reporter"
	WarnSkippingHostType   = "skipping excluded host type"
	WarnPayloadTracker     = "unable to send message to payload tracker"
	ErrorNoAccountProvided = "no account provided in host message"
	ErrorKafkaSend         = "unable to send evaluation message"
	ErrorProcessUpload     = "unable to process upload"
	UploadSuccessNoEval    = "upload event handled successfully, no eval required"
	UploadSuccess          = "upload event handled successfully"
	FlushedFullBuffer      = "flushing full eval event buffer"
	FlushedTimeoutBuffer   = "flushing eval event buffer after timeout"
	ErrorUnmarshalMetadata = "unable to unmarshall platform metadata value"
	ErrorStatus            = "error"
	SuccessStatus          = "success"
)

var DeletionThreshold = time.Hour * time.Duration(utils.GetIntEnvOrDefault("SYSTEM_DELETE_HRS", 4))

type Host struct {
	ID                    string                  `json:"id,omitempty"`
	DisplayName           *string                 `json:"display_name,omitempty"`
	Account               *string                 `json:"account,omitempty"`
	OrgID                 *string                 `json:"org_id,omitempty"`
	StaleTimestamp        *types.Rfc3339Timestamp `json:"stale_timestamp,omitempty"`
	StaleWarningTimestamp *types.Rfc3339Timestamp `json:"stale_warning_timestamp,omitempty"`
	CulledTimestamp       *types.Rfc3339Timestamp `json:"culled_timestamp,omitempty"`
	Reporter              string                  `json:"reporter,omitempty"`
	SystemProfile         inventory.SystemProfile `json:"system_profile,omitempty"`
}

type HostMetadata struct {
	RequestID string `json:"request_id"`
}

type HostEvent struct {
	Type             string               `json:"type"`
	PlatformMetadata HostPlatformMetadata `json:"platform_metadata"`
	Host             Host                 `json:"host"`
	Metadata         HostMetadata         `json:"metadata"`
}

type HostPlatformMetadata struct {
	CustomMetadata HostCustomMetadata `json:"custom_metadata,omitempty"`
}
type HostCustomMetadata struct {
	YumUpdates json.RawMessage `json:"yum_updates,omitempty"`
}

//nolint:funlen
func HandleUpload(event HostEvent) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageHandlingDuration.WithLabelValues(EventUpload))

	updateReporterCounter(event.Host.Reporter)

	payloadTrackerEvent := mqueue.PayloadTrackerEvent{
		Account:     event.Host.Account,
		OrgID:       event.Host.OrgID,
		RequestID:   &event.Metadata.RequestID,
		InventoryID: event.Host.ID,
		Status:      "received",
	}

	if _, ok := excludedReporters[event.Host.Reporter]; ok {
		utils.Log("inventoryID", event.Host.ID, "reporter", event.Host.Reporter).Warn(WarnSkippingReporter)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedWarnExcludedReporter).Inc()
		utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("message-skip"))
		sendPayloadStatus(ptWriter, payloadTrackerEvent, "", WarnSkippingReporter)
		return nil
	}

	if _, ok := excludedHostTypes[event.Host.SystemProfile.HostType]; ok {
		utils.Log("inventoryID", event.Host.ID, "hostType", event.Host.SystemProfile.HostType).Warn(WarnSkippingHostType)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedWarnExcludedHostType).Inc()
		utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("message-skip"))
		sendPayloadStatus(ptWriter, payloadTrackerEvent, "", WarnSkippingHostType)
		return nil
	}

	if (event.Host.Account == nil || *event.Host.Account == "") && (event.Host.OrgID == nil || *event.Host.OrgID == "") {
		utils.Log("inventoryID", event.Host.ID).Error(ErrorNoAccountProvided)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorIdentity).Inc()
		utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("message-skip"))
		sendPayloadStatus(ptWriter, payloadTrackerEvent, "", ErrorNoAccountProvided)
		return nil
	}

	sendPayloadStatus(ptWriter, payloadTrackerEvent, "", "")
	yumUpdates, err := getYumUpdates(event)
	if err != nil {
		// don't fail, use vmaas evaluation
		utils.Log("err", err).Error("Could not get yum updates")
	}
	utils.Log("inventoryID", event.Host.ID, "yum_updates", string(yumUpdates)).Trace()

	if len(event.Host.SystemProfile.GetInstalledPackages()) == 0 && yumUpdates == nil {
		utils.Log("inventoryID", event.Host.ID).Warn(WarnSkippingNoPackages)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedWarnNoPackages).Inc()
		utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues(ReceivedWarnNoPackages))
		sendPayloadStatus(ptWriter, payloadTrackerEvent, ErrorStatus, WarnSkippingNoPackages)
		return nil
	}

	sys, err := processUpload(&event.Host, yumUpdates)

	if err != nil {
		utils.Log("inventoryID", event.Host.ID, "err", err.Error()).Error(ErrorProcessUpload)
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedErrorProcessing).Inc()
		utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues(ReceivedErrorProcessing))
		sendPayloadStatus(ptWriter, payloadTrackerEvent, ErrorStatus, ErrorProcessUpload)
		return errors.Wrap(err, "Could not process upload")
	}

	// Deleted system, return nil
	if sys == nil {
		messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedDeleted).Inc()
		utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues(ReceivedDeleted))
		sendPayloadStatus(ptWriter, payloadTrackerEvent, SuccessStatus, ReceivedDeleted)
		return nil
	}

	if sys.UnchangedSince != nil && sys.LastEvaluation != nil {
		if sys.UnchangedSince.Before(*sys.LastEvaluation) {
			messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedSuccessNoEval).Inc()
			utils.Log("inventoryID", event.Host.ID).Info(UploadSuccessNoEval)
			utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues(ReceivedSuccessNoEval))
			sendPayloadStatus(ptWriter, payloadTrackerEvent, SuccessStatus, ReceivedSuccessNoEval)
			return nil
		}
	}

	bufferEvalEvents(sys.InventoryID, sys.RhAccountID, &payloadTrackerEvent)

	messagesReceivedCnt.WithLabelValues(EventUpload, ReceivedSuccess).Inc()
	utils.Log("inventoryID", event.Host.ID).Info(UploadSuccess)
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues(ReceivedSuccess))
	return nil
}

func sendPayloadStatus(w mqueue.Writer, event mqueue.PayloadTrackerEvent, status string, statusMsg string) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("payload-tracker-status"))
	if status != "" {
		event.Status = status
	}
	if statusMsg != "" {
		event.StatusMsg = statusMsg
	}
	if err := mqueue.SendMessages(base.Context, w, &event); err != nil {
		utils.Log("err", err.Error()).Warn(WarnPayloadTracker)
	}
}

// accumulate events and create group PlatformEvents to save some resources
var evalBufferSize = 5 * mqueue.BatchSize

var evalBuffer = make(mqueue.EvalDataSlice, 0, evalBufferSize+1)
var ptBuffer = make(mqueue.PayloadTrackerEvents, 0, evalBufferSize+1)
var flushTimer = time.AfterFunc(87600*time.Hour, func() {
	utils.Log().Info(FlushedTimeoutBuffer)
	flushEvalEvents()
})

// send events after full buffer or timeout
func bufferEvalEvents(inventoryID string, rhAccountID int, ptEvent *mqueue.PayloadTrackerEvent) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-eval-events"))
	evalData := mqueue.EvalData{
		InventoryID: inventoryID,
		RhAccountID: rhAccountID,
		AccountInfo: mqueue.AccountInfo{
			AccountName: ptEvent.Account,
			OrgID:       ptEvent.OrgID,
		},
		RequestID: *ptEvent.RequestID,
	}
	evalBuffer = append(evalBuffer, evalData)
	ptBuffer = append(ptBuffer, *ptEvent)
	flushTimer.Reset(uploadEvalTimeout)
	if len(evalBuffer) >= evalBufferSize {
		utils.Log().Info(FlushedFullBuffer)
		flushEvalEvents()
	}
}

func flushEvalEvents() {
	tStart := time.Now()
	err := mqueue.SendMessages(base.Context, evalWriter, &evalBuffer)
	if err != nil {
		utils.Log("err", err.Error()).Error(ErrorKafkaSend)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-evaluator"))
	err = mqueue.SendMessages(base.Context, ptWriter, &ptBuffer)
	if err != nil {
		utils.Log("err", err.Error()).Warn(WarnPayloadTracker)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-payload-tracker"))
	// empty buffer
	evalBuffer = evalBuffer[:0]
	ptBuffer = ptBuffer[:0]
}

func updateReporterCounter(reporter string) {
	if _, ok := validReporters[reporter]; ok {
		receivedFromReporter.WithLabelValues(reporter).Inc()
	} else {
		receivedFromReporter.WithLabelValues("unknown").Inc()
		utils.Log("reporter", reporter).Warn("unknown reporter")
	}
}

// nolint: funlen
// Stores or updates base system profile, returing internal system id
func updateSystemPlatform(tx *gorm.DB, inventoryID string, accountID int, host *Host,
	yumUpdates []byte, updatesReq *vmaas.UpdatesV3Request) (*models.SystemPlatform, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("update-system-platform"))
	updatesReqJSON, err := json.Marshal(updatesReq)
	if err != nil {
		return nil, errors.Wrap(err, "Serializing vmaas request")
	}

	hash := sha256.Sum256(updatesReqJSON)
	jsonChecksum := hex.EncodeToString(hash[:])
	hash = sha256.Sum256(yumUpdates)
	yumChecksum := hex.EncodeToString(hash[:])

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
	updatesReqJSONString := string(updatesReqJSON)
	systemPlatform := models.SystemPlatform{
		InventoryID:           inventoryID,
		RhAccountID:           accountID,
		DisplayName:           displayName,
		VmaasJSON:             utils.EmptyToNil(&updatesReqJSONString),
		JSONChecksum:          utils.EmptyToNil(&jsonChecksum),
		LastUpload:            &now,
		StaleTimestamp:        host.StaleTimestamp.Time(),
		StaleWarningTimestamp: host.StaleWarningTimestamp.Time(),
		CulledTimestamp:       host.CulledTimestamp.Time(),
		Stale:                 staleWarning != nil && staleWarning.Before(time.Now()),
		ReporterID:            getReporterID(host.Reporter),
		YumUpdates:            yumUpdates,
	}

	var oldChecksums map[string]string
	// Lock the row for update & return checksum
	tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Table("system_platform").
		Where("inventory_id = ?::uuid", inventoryID).
		Where("rh_account_id = ?", accountID).
		First("json_checksum, digest(yum_updates,'sha256') as yum_checksum", &oldChecksums)

	shouldUpdateRepos := false
	var addedRepos, addedSysRepos, deletedSysRepos int64

	// Skip updating vmaas_json if the checksum haven't changed. Should reduce TOAST trashing
	if oldChecksums["json_checksum"] != jsonChecksum {
		colsToUpdate = append(colsToUpdate, "vmaas_json", "json_checksum", "reporter_id")
		shouldUpdateRepos = true
	}

	// Skip updating yum_updates if the checksum haven't changed.
	if oldChecksums["yum_checksum"] != yumChecksum {
		colsToUpdate = append(colsToUpdate, "yum_updates")
	}

	if err := database.OnConflictUpdateMulti(tx, []string{"rh_account_id", "inventory_id"}, colsToUpdate...).
		Save(&systemPlatform).Error; err != nil {
		return nil, errors.Wrap(err, "Unable to save or update system in database")
	}

	if shouldUpdateRepos {
		// We also don't need to update repos if vmaas_json haven't changed
		addedRepos, addedSysRepos, deletedSysRepos, err = updateRepos(tx, host.SystemProfile, accountID,
			systemPlatform.ID, updatesReq.GetRepositoryList())
		if err != nil {
			utils.Log("repository_list", updatesReq.RepositoryList, "inventoryID", systemPlatform.ID).
				Error("repos failed to insert")
			return nil, errors.Wrap(err, "unable to update system repos")
		}
	}

	utils.Log("inventoryID", inventoryID, "packages", len(updatesReq.PackageList), "repos",
		len(updatesReq.GetRepositoryList()), "modules", len(updatesReq.GetModulesList()),
		"addedRepos", addedRepos, "addedSysRepos", addedSysRepos, "deletedSysRepos", deletedSysRepos).
		Info("System created or updated successfully")
	return &systemPlatform, nil
}

func getReporterID(reporter string) *int {
	if id, ok := validReporters[reporter]; ok {
		return &id
	}
	utils.Log("reporter", reporter).Warn("no reporter id found, returning nil")
	return nil
}

// EPEL uses the `epel` repo identifier on both rhel 7 and rhel 8. We create our own mapping to
// `epel-7` and `epel-8`
func fixEpelRepos(sys *inventory.SystemProfile, repos []string) []string {
	if sys == nil || sys.OperatingSystem.Major == 0 {
		return repos
	}

	for i, r := range repos {
		if r == "epel" {
			repos[i] = fmt.Sprintf("%s-%d", r, sys.OperatingSystem.Major)
		}
	}
	return repos
}

func updateRepos(tx *gorm.DB, profile inventory.SystemProfile, rhAccountID int,
	systemID int, repos []string) (addedRepos int64, addedSysRepos int64, deletedSysRepos int64, err error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("update-repos"))
	repos = fixEpelRepos(&profile, repos)
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
	if len(repos) == 0 {
		return repoIDs, 0, nil
	}
	repoObjs := make(models.RepoSlice, len(repos))
	for i, repo := range repos {
		repoObjs[i] = models.Repo{Name: repo}
	}

	txOnConflict := tx.Clauses(clause.OnConflict{
		DoNothing: true,
	})
	err = txOnConflict.Create(repoObjs).Error
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

	txOnConflict := tx.Clauses(clause.OnConflict{
		DoNothing: true,
	})
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

func processRepos(systemProfile *inventory.SystemProfile) *[]string {
	repos := make([]string, 0, len(systemProfile.GetYumRepos()))
	for _, r := range systemProfile.GetYumRepos() {
		rID := r.ID
		if len(strings.TrimSpace(rID)) == 0 {
			utils.Log("repo", rID).Warn("removed repo with invalid name")
			continue
		}

		if r.Enabled {
			repos = append(repos, rID)
		}
	}
	fixEpelRepos(systemProfile, repos)
	return &repos
}

func processModules(systemProfile *inventory.SystemProfile) *[]vmaas.UpdatesV3RequestModulesList {
	var modules []vmaas.UpdatesV3RequestModulesList
	if count := len(systemProfile.GetDnfModules()); count > 0 {
		modules = make([]vmaas.UpdatesV3RequestModulesList, count)
		for i, m := range systemProfile.GetDnfModules() {
			modules[i] = vmaas.UpdatesV3RequestModulesList{
				ModuleName:   m.Name,
				ModuleStream: m.Stream,
			}
		}
	}
	return &modules
}

// We have received new upload, update stored host data, and re-evaluate the host against VMaaS
func processUpload(host *Host, yumUpdates []byte) (*models.SystemPlatform, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("upload-processing"))
	// Ensure we have account stored
	var accountNumber string
	var orgID string
	if host.Account != nil {
		accountNumber = *host.Account
	}
	if host.OrgID != nil {
		orgID = *host.OrgID
	}
	identity := utils.Identity{
		AccountNumber: accountNumber,
		OrgID:         orgID,
	}
	accountID, err := middlewares.GetOrCreateAccount(&identity)
	if err != nil {
		return nil, errors.Wrap(err, "saving account into the database")
	}

	systemProfile := host.SystemProfile
	// Prepare VMaaS request
	updatesReq := vmaas.UpdatesV3Request{
		PackageList:    systemProfile.GetInstalledPackages(),
		RepositoryList: processRepos(&systemProfile),
		ModulesList:    processModules(&systemProfile),
		Basearch:       systemProfile.Arch,
		SecurityOnly:   utils.PtrBool(false),
		LatestOnly:     utils.PtrBool(true),
	}

	// use rhsm version if set
	releasever := systemProfile.Rhsm.Version
	if len(releasever) > 0 {
		updatesReq.SetReleasever(releasever)
	}

	tx := database.Db.WithContext(base.Context).Begin()
	defer tx.Rollback()

	var deleted models.DeletedSystem
	if err := tx.Find(&deleted, "inventory_id = ?", host.ID).Error; err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrap(err, "Checking deleted systems")
	}

	// If the system was deleted in last hour, don't register this upload
	if deleted.InventoryID != "" && deleted.WhenDeleted.After(time.Now().Add(-DeletionThreshold)) {
		utils.Log("inventoryID", host.ID).Info("Received recently deleted system")
		return nil, nil
	}
	sys, err := updateSystemPlatform(tx, host.ID, accountID, host, yumUpdates, &updatesReq)
	if err != nil {
		return nil, errors.Wrap(err, "saving system into the database")
	}
	err = tx.Commit().Error
	if err != nil {
		return nil, errors.Wrap(err, "Committing changes")
	}
	return sys, nil
}

func getYumUpdates(event HostEvent) ([]byte, error) {
	var parsed vmaas.UpdatesV2Response
	yumUpdates := event.PlatformMetadata.CustomMetadata.YumUpdates

	err := json.Unmarshal(yumUpdates, &parsed)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshall yum updates")
	}

	updatesMap := parsed.GetUpdateList()
	if len(updatesMap) == 0 {
		// system does not have any yum updates
		return yumUpdates, nil
	}
	// we need to get all packages to show up-to-date packages
	installedPkgs := event.Host.SystemProfile.GetInstalledPackages()
	for _, pkg := range installedPkgs {
		if _, has := updatesMap[pkg]; !has {
			updatesMap[pkg] = vmaas.UpdatesV2ResponseUpdateList{}
		}
	}
	parsed.UpdateList = &updatesMap

	if err = utils.RemoveNonLatestPackages(&parsed); err != nil {
		return nil, errors.Wrap(err, "couldn't remove non-latest packages")
	}
	yumUpdates, err = json.Marshal(parsed)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshall yum updates")
	}
	return yumUpdates, nil
}
