package listener

import (
	"app/base"
	"app/base/api"
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
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	stdErrors "errors"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	WarnPayloadTracker        = "unable to send message to payload tracker"
	ErrorNoAccountProvided    = "no account provided in host message"
	ErrorKafkaSend            = "unable to send evaluation message"
	ErrorProcessUpload        = "unable to process upload"
	UploadSuccessNoEval       = "upload event handled successfully, no eval required"
	UploadSuccess             = "upload event handled successfully"
	DeleteSuccess             = "delete event handled successfully"
	FlushedFullBuffer         = "flushing full eval event buffer"
	FlushedTimeoutBuffer      = "flushing eval event buffer after timeout"
	ErrorUnmarshalMetadata    = "unable to unmarshall platform metadata value"
	ErrorStatus               = "error"
	ProcessingStatus          = "processing"
	ReceivedStatus            = "received"
	SuccessStatus             = "success"
	RhuiPathPart              = "/rhui/"
	RepoPathPattern           = "(/content/.*)"
	RepoBasearchPlaceholder   = "$basearch"
	RepoReleaseverPlaceholder = "$releasever"
)

var (
	ErrNoPackages        = errors.New("skipping profile with no packages")
	ErrReporter          = errors.New("skipping excluded reporter")
	ErrHostType          = errors.New("skipping excluded host type")
	ErrBadPackages       = errors.New("skipping profile with malformed packages")
	ErrNoAccountProvided = errors.New("no account provided in host message")
	ErrKafkaSend         = errors.New("unable to send evaluation message")
	ErrProcessUpload     = errors.New("unable to process upload")
)

var (
	repoPathRegex = regexp.MustCompile(RepoPathPattern)
	spacesRegex   = regexp.MustCompile(`^\s*$`)
	httpClient    *api.Client
	metricByErr   = map[error]string{
		ErrNoPackages:        ReceivedWarnNoPackages,
		ErrReporter:          ReceivedWarnExcludedReporter,
		ErrHostType:          ReceivedWarnExcludedHostType,
		ErrBadPackages:       ReceivedWarnBadPackages,
		ErrNoAccountProvided: ReceivedErrorIdentity,
		ErrProcessUpload:     ReceivedErrorProcessing,
	}
)

type Host struct {
	ID                    string                  `json:"id,omitempty"`
	DisplayName           *string                 `json:"display_name,omitempty"`
	OrgID                 *string                 `json:"org_id,omitempty"`
	StaleTimestamp        *types.Rfc3339Timestamp `json:"stale_timestamp,omitempty"`
	StaleWarningTimestamp *types.Rfc3339Timestamp `json:"stale_warning_timestamp,omitempty"`
	CulledTimestamp       *types.Rfc3339Timestamp `json:"culled_timestamp,omitempty"`
	Reporter              string                  `json:"reporter,omitempty"`
	SystemProfile         inventory.SystemProfile `json:"system_profile,omitempty"`
	ParsedYumUpdates      *YumUpdates             `json:"-"`
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
	YumUpdates      json.RawMessage `json:"yum_updates,omitempty"`
	YumUpdatesS3URL *string         `json:"yum_updates_s3url,omitempty"`
}

type YumUpdates struct {
	RawParsed     json.RawMessage
	BuiltPkgcache bool
}

// GetRawParsed returns prepared parsed raw yumupdates
func (y *YumUpdates) GetRawParsed() json.RawMessage {
	if y == nil {
		return nil
	}
	return y.RawParsed
}

// GetBuiltPkgcache returns boolean for build_pkgcache from yum_updates
func (y *YumUpdates) GetBuiltPkgcache() bool {
	if y == nil {
		return false
	}
	return y.BuiltPkgcache
}

func HandleUpload(event HostEvent) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageHandlingDuration.WithLabelValues(EventUpload))

	httpClient = &api.Client{
		HTTPClient: &http.Client{},
		Debug:      useTraceLevel,
	}
	updateReporterCounter(event.Host.Reporter)

	ptEvent := mqueue.PayloadTrackerEvent{
		OrgID:       event.Host.OrgID,
		RequestID:   &event.Metadata.RequestID,
		InventoryID: event.Host.ID,
		Status:      ReceivedStatus,
	}

	if err := validateHost(&event.Host); err != nil {
		return handleListenerErrors(err, &event, &ptEvent, tStart, ReceivedStatus)
	}

	sendPayloadStatus(ptWriter, ptEvent, "", "Received by listener")
	yumUpdates, err := getYumUpdates(event, httpClient)
	if err != nil {
		// don't fail, use vmaas evaluation
		utils.LogError("err", err, "Could not get yum updates")
	}

	sys, err := processUpload(&event.Host, yumUpdates)
	if err != nil {
		return handleListenerErrors(stdErrors.Join(ErrProcessUpload, err), &event, &ptEvent, tStart, ErrorStatus)
	}
	// Deleted system, return nil
	if sys == nil {
		logAndObserve(DeleteSuccess, ReceivedDeleted, &event, &ptEvent, tStart, SuccessStatus, true)
		return nil
	}

	if sys.UnchangedSince != nil && sys.LastEvaluation != nil {
		if sys.UnchangedSince.Before(*sys.LastEvaluation) {
			logAndObserve(UploadSuccessNoEval, ReceivedSuccessNoEval, &event, &ptEvent, tStart, SuccessStatus, true)
			return nil
		}
	}

	ptEvent.StatusMsg = ProcessingStatus
	bufferEvalEvents(sys.InventoryID, sys.RhAccountID, &ptEvent)
	logAndObserve(UploadSuccess, ReceivedSuccess, &event, &ptEvent, tStart, SuccessStatus, false)
	return nil
}

func handleListenerErrors(err error, event *HostEvent, ptEvent *mqueue.PayloadTrackerEvent,
	tStart time.Time, status string) error {
	if err != nil {
		if errors.Is(err, base.ErrFatal) {
			// fatal error which should restart the pod
			utils.LogError("inventoryID", event.Host.ID, "err", err.Error())
		} else {
			utils.LogWarn("inventoryID", event.Host.ID, "reporter", event.Host.Reporter,
				"hostType", event.Host.SystemProfile.HostType, "err", err.Error())
		}
	}
	metric, ok := metricByErr[err]
	if !ok {
		metric = "internal-error"
	}
	logAndObserve(err.Error(), metric, event, ptEvent, tStart, status, true)

	return err
}

func logAndObserve(msg, metric string, event *HostEvent, ptEvent *mqueue.PayloadTrackerEvent,
	tStart time.Time, status string, withPayloadTracker bool) {
	eventMsgsReceivedCnt.WithLabelValues(EventUpload, metric).Inc()
	utils.LogInfo("inventoryID", event.Host.ID, msg)
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues(metric))
	if withPayloadTracker {
		sendPayloadStatus(ptWriter, *ptEvent, status, metric)
	}
}

func validateHost(host *Host) error {
	if _, ok := allowedReporters[host.Reporter]; !ok {
		return ErrReporter
	}
	if host.OrgID == nil || *host.OrgID == "" {
		return ErrNoAccountProvided
	}

	if _, ok := excludedHostTypes[host.SystemProfile.HostType]; ok {
		return ErrHostType
	}

	installedPackages := host.SystemProfile.GetInstalledPackages()
	if len(installedPackages) == 0 {
		return ErrNoPackages
	}
	if err := checkPackagesEpoch(installedPackages); err != nil {
		err = stdErrors.Join(ErrBadPackages, err)
		return err
	}
	return nil
}

func checkPackagesEpoch(packages []string) error {
	if len(packages) > 0 {
		// parse first package from list and skip upload if pkg is malformed, e.g. without epoch
		if _, err := utils.ParseNevra(packages[0]); err != nil {
			return err
		}
	}
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
		utils.LogWarn("err", err.Error(), WarnPayloadTracker)
	}
}

// accumulate events and create group PlatformEvents to save some resources
var evalBufferSize = 5 * mqueue.BatchSize
var eBuffer = struct {
	EvalBuffer mqueue.EvalDataSlice
	PtBuffer   mqueue.PayloadTrackerEvents
	Lock       sync.Mutex
}{
	EvalBuffer: make(mqueue.EvalDataSlice, 0, evalBufferSize+1),
	PtBuffer:   make(mqueue.PayloadTrackerEvents, 0, evalBufferSize+1),
	Lock:       sync.Mutex{},
}
var flushTimer = time.AfterFunc(87600*time.Hour, func() {
	utils.LogInfo(FlushedTimeoutBuffer)
	flushEvalEvents()
})

// send events after full buffer or timeout
func bufferEvalEvents(inventoryID string, rhAccountID int, ptEvent *mqueue.PayloadTrackerEvent) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-eval-events"))

	eBuffer.Lock.Lock()
	evalData := mqueue.EvalData{
		InventoryID: inventoryID,
		RhAccountID: rhAccountID,
		OrgID:       ptEvent.OrgID,
		RequestID:   *ptEvent.RequestID,
	}
	eBuffer.EvalBuffer = append(eBuffer.EvalBuffer, evalData)
	eBuffer.PtBuffer = append(eBuffer.PtBuffer, *ptEvent)
	eBuffer.Lock.Unlock()

	flushTimer.Reset(uploadEvalTimeout)
	if len(eBuffer.EvalBuffer) >= evalBufferSize {
		utils.LogInfo(FlushedFullBuffer)
		flushEvalEvents()
	}
}

func flushEvalEvents() {
	tStart := time.Now()
	eBuffer.Lock.Lock()
	defer eBuffer.Lock.Unlock()
	err := mqueue.SendMessages(base.Context, evalWriter, eBuffer.EvalBuffer)
	if err != nil {
		utils.LogError("err", err.Error(), ErrorKafkaSend)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-evaluator"))
	err = mqueue.SendMessages(base.Context, ptWriter, eBuffer.PtBuffer)
	if err != nil {
		utils.LogWarn("err", err.Error(), WarnPayloadTracker)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-payload-tracker"))
	// empty buffer
	eBuffer.EvalBuffer = eBuffer.EvalBuffer[:0]
	eBuffer.PtBuffer = eBuffer.PtBuffer[:0]
}

func updateReporterCounter(reporter string) {
	if _, ok := validReporters[reporter]; ok {
		receivedFromReporter.WithLabelValues(reporter).Inc()
	} else {
		receivedFromReporter.WithLabelValues("unknown").Inc()
		utils.LogWarn("reporter", reporter, "unknown reporter")
	}
}

// nolint: funlen
// Stores or updates base system profile, returing internal system id
func updateSystemPlatform(tx *gorm.DB, inventoryID string, accountID int, host *Host,
	yumUpdates *YumUpdates, updatesReq *vmaas.UpdatesV3Request) (*models.SystemPlatform, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("update-system-platform"))
	updatesReqJSON, err := json.Marshal(updatesReq)
	if err != nil {
		return nil, errors.Wrap(err, "Serializing vmaas request")
	}

	hash := sha256.Sum256(updatesReqJSON)
	jsonChecksum := hex.EncodeToString(hash[:])
	hash = sha256.Sum256(yumUpdates.GetRawParsed())
	yumChecksum := hex.EncodeToString(hash[:])

	var colsToUpdate = []string{
		"display_name",
		"last_upload",
		"stale",
		"stale_timestamp",
		"stale_warning_timestamp",
		"culled_timestamp",
		"satellite_managed",
		"built_pkgcache",
		"arch",
		"bootc",
	}

	now := time.Now()
	displayName := inventoryID
	if host.DisplayName != nil && len(*host.DisplayName) > 0 && !spacesRegex.MatchString(*host.DisplayName) {
		displayName = *host.DisplayName
	}

	isBootc := false
	if len(host.SystemProfile.BootcStatus.Booted.Image) > 0 {
		isBootc = true
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
		YumUpdates:            yumUpdates.GetRawParsed(),
		YumChecksum:           utils.EmptyToNil(&yumChecksum),
		SatelliteManaged:      host.SystemProfile.SatelliteManaged,
		BuiltPkgcache:         yumUpdates.GetBuiltPkgcache(),
		Arch:                  host.SystemProfile.Arch,
		Bootc:                 isBootc,
	}

	type OldChecksums struct {
		JSONChecksum string `gorm:"column:json_checksum"`
		YumChecksum  string `gorm:"column:yum_checksum"`
	}
	var oldChecksums OldChecksums
	// Lock the row for update & return checksum
	tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(models.SystemPlatform{}).
		Where("inventory_id = ?::uuid", inventoryID).
		Where("rh_account_id = ?", accountID).
		Select("json_checksum, yum_checksum").
		First(&oldChecksums)

	shouldUpdateRepos := false
	var addedRepos, addedSysRepos, deletedSysRepos int64

	// Skip updating vmaas_json if the checksum haven't changed. Should reduce TOAST trashing
	if oldChecksums.JSONChecksum != jsonChecksum {
		colsToUpdate = append(colsToUpdate, "vmaas_json", "json_checksum", "reporter_id")
		shouldUpdateRepos = true
	}

	// Skip updating yum_updates if the checksum haven't changed.
	if oldChecksums.YumChecksum != yumChecksum {
		colsToUpdate = append(colsToUpdate, "yum_updates", "yum_checksum")
	}

	if err := storeOrUpdateSysPlatform(tx, &systemPlatform, colsToUpdate); err != nil {
		return nil, errors.Wrap(err, "Unable to save or update system in database")
	}

	if shouldUpdateRepos {
		// We also don't need to update repos if vmaas_json haven't changed
		addedRepos, addedSysRepos, deletedSysRepos, err = updateRepos(tx, host.SystemProfile, accountID,
			systemPlatform.ID, updatesReq.RepositoryList)
		if err != nil {
			utils.LogError("repository_list", updatesReq.RepositoryList, "inventoryID", systemPlatform.ID,
				"repos failed to insert")
			return nil, errors.Wrap(err, "unable to update system repos")
		}
	}

	utils.LogInfo("inventoryID", inventoryID, "packages", len(updatesReq.PackageList), "repos",
		len(updatesReq.RepositoryList), "modules", len(updatesReq.GetModulesList()),
		"addedRepos", addedRepos, "addedSysRepos", addedSysRepos, "deletedSysRepos", deletedSysRepos,
		"System created or updated successfully")
	return &systemPlatform, nil
}

func storeOrUpdateSysPlatform(tx *gorm.DB, system *models.SystemPlatform, colsToUpdate []string) error {
	if err := tx.Where("rh_account_id = ? AND inventory_id = ?", system.RhAccountID, system.InventoryID).
		Select("id").Find(system).Error; err != nil {
		utils.LogWarn("err", err, "couldn't find system for update")
	}

	// return system_platform record after update
	tx = tx.Clauses(clause.Returning{
		Columns: []clause.Column{
			{Name: "id"}, {Name: "inventory_id"}, {Name: "rh_account_id"},
			{Name: "unchanged_since"}, {Name: "last_evaluation"},
		},
	})

	if system.ID != 0 {
		// update system
		err := tx.Select(colsToUpdate).Updates(system).Error
		return base.WrapFatalDBError(err, "unable to update system_platform")
	}
	// insert system
	err := database.OnConflictUpdateMulti(tx, []string{"rh_account_id", "inventory_id"}, colsToUpdate...).
		Save(system).Error
	return base.WrapFatalDBError(err, "unable to insert to system_platform")
}

func getReporterID(reporter string) *int {
	if id, ok := validReporters[reporter]; ok {
		return &id
	}
	utils.LogWarn("reporter", reporter, "no reporter id found, returning nil")
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
	systemID int64, repos []string) (addedRepos int64, addedSysRepos int64, deletedSysRepos int64, err error) {
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

func ensureReposInDB(tx *gorm.DB, repos []string) (repoIDs []int64, added int64, err error) {
	if len(repos) == 0 {
		return repoIDs, 0, nil
	}
	repoIDs = make([]int64, 0, len(repos))

	var existingRepos models.RepoSlice
	err = tx.Model(&models.Repo{}).Where("name IN (?)", repos).Find(&existingRepos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.Wrapf(err, "couldn't find repos: %s", repos)
		}
		return nil, 0, base.WrapFatalDBError(err, "unable to load repos")
	}

	inDBIDs := make(map[string]int64)
	for _, er := range existingRepos {
		inDBIDs[er.Name] = er.ID
	}

	toStore := make(models.RepoSlice, 0, len(repos))
	for _, repo := range repos {
		if id, has := inDBIDs[repo]; has {
			repoIDs = append(repoIDs, id)
		} else {
			toStore = append(toStore, models.Repo{Name: repo})
		}
	}

	if len(toStore) > 0 {
		txOnConflict := tx.Clauses(clause.OnConflict{
			DoNothing: true,
		})
		err = txOnConflict.Create(&toStore).Error
		if err != nil {
			return nil, 0, base.WrapFatalDBError(err, "unable to update repos")
		}
		added = txOnConflict.RowsAffected
		for _, repo := range toStore {
			repoIDs = append(repoIDs, repo.ID)
		}
	}
	reposAddedCnt.Add(float64(added))

	return repoIDs, added, nil
}

func updateSystemRepos(tx *gorm.DB, rhAccountID int, systemID int64, repoIDs []int64) (
	nAdded int64, nDeleted int64, err error) {
	repoSystemObjs := make(models.SystemRepoSlice, len(repoIDs))
	for i, repoID := range repoIDs {
		repoSystemObjs[i] = models.SystemRepo{RhAccountID: int64(rhAccountID), SystemID: systemID, RepoID: repoID}
	}

	txOnConflict := tx.Clauses(clause.OnConflict{
		DoNothing: true,
	})
	err = database.BulkInsert(txOnConflict, repoSystemObjs)
	if err != nil {
		return 0, 0, base.WrapFatalDBError(err, "unable to update system repos")
	}
	nAdded = txOnConflict.RowsAffected

	nDeleted, err = deleteOtherSystemRepos(tx, rhAccountID, systemID, repoIDs)
	if err != nil {
		return nAdded, 0, base.WrapFatalDBError(err, "unable to delete out-of-date system repos")
	}

	return nAdded, nDeleted, nil
}

func deleteOtherSystemRepos(tx *gorm.DB, rhAccountID int, systemID int64, repoIDs []int64) (nDeleted int64, err error) {
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

// get rhui repository path form `system_profile.yum_repos.mirrorlist` or `system_profile.yum_repos.base_url`
func getRepoPath(systemProfile *inventory.SystemProfile, repo *inventory.YumRepo) (string, error) {
	var repoPath string
	if systemProfile == nil || repo == nil {
		return repoPath, nil
	}
	if len(repo.Mirrorlist) == 0 && len(repo.BaseURL) == 0 {
		return repoPath, nil
	}

	repoURL := repo.Mirrorlist
	if len(repoURL) == 0 {
		repoURL = repo.BaseURL
	}

	url, err := url.Parse(repoURL)
	if err != nil {
		return repoPath, errors.Wrap(err, "couldn't parse repo mirrorlist or base_url")
	}

	foundRepoPath := repoPathRegex.FindString(url.Path)
	if strings.Contains(foundRepoPath, RhuiPathPart) {
		repoPath = foundRepoPath
		if systemProfile.Arch != nil {
			repoPath = strings.ReplaceAll(repoPath, RepoBasearchPlaceholder, *systemProfile.Arch)
		}
		if systemProfile.Releasever != nil {
			repoPath = strings.ReplaceAll(repoPath, RepoReleaseverPlaceholder, *systemProfile.Releasever)
		}
		return repoPath, nil
	}
	return repoPath, nil
}

func processRepos(systemProfile *inventory.SystemProfile) ([]string, []string) {
	yumRepos := systemProfile.GetYumRepos()
	seen := make(map[string]bool, len(yumRepos))
	repos := make([]string, 0, len(yumRepos))
	repoPaths := make([]string, 0, len(yumRepos))
	for _, r := range yumRepos {
		r := r
		rID := r.ID
		if seen[rID] {
			// remove duplicate repos
			continue
		}
		seen[rID] = true
		if len(strings.TrimSpace(rID)) == 0 {
			utils.LogWarn("repo", rID, "removed repo with invalid name")
			continue
		}

		if r.Enabled {
			repos = append(repos, rID)
			repoPath, err := getRepoPath(systemProfile, &r)
			if err != nil {
				utils.LogWarn("repo", rID, "mirrorlist", r.Mirrorlist, "base_url", r.BaseURL, "invalid repository_path")
			}
			if len(repoPath) > 0 {
				repoPaths = append(repoPaths, repoPath)
			}
		}
	}
	fixEpelRepos(systemProfile, repos)
	return repos, repoPaths
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
func processUpload(host *Host, yumUpdates *YumUpdates) (*models.SystemPlatform, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("upload-processing"))
	// Ensure we have account stored
	accountID, err := middlewares.GetOrCreateAccount(host.GetOrgID())
	if err != nil {
		return nil, errors.Wrap(err, "saving account into the database")
	}

	systemProfile := host.SystemProfile
	repos, repoPaths := processRepos(&systemProfile)
	// Prepare VMaaS request
	updatesReq := vmaas.UpdatesV3Request{
		PackageList:     systemProfile.GetInstalledPackages(),
		RepositoryList:  repos,
		RepositoryPaths: repoPaths,
		ModulesList:     processModules(&systemProfile),
		Basearch:        systemProfile.Arch,
		SecurityOnly:    utils.PtrBool(false),
		LatestOnly:      utils.PtrBool(true),
	}

	// use rhsm version if set
	releasever := systemProfile.Rhsm.Version
	if releasever == "" && systemProfile.Releasever != nil {
		releasever = *systemProfile.Releasever
	}
	if len(releasever) > 0 {
		updatesReq.SetReleasever(releasever)
	}

	tx := database.DB.WithContext(base.Context).Begin()
	defer tx.Rollback()

	var deleted models.DeletedSystem
	if err := tx.Find(&deleted, "inventory_id = ?", host.ID).Error; err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, base.WrapFatalDBError(err, "checking deleted systems")
	}

	// If the system was deleted in last hour, don't register this upload
	if deleted.InventoryID != "" && deleted.WhenDeleted.After(time.Now().Add(-deletionThreshold)) {
		utils.LogInfo("inventoryID", host.ID, "Received recently deleted system")
		return nil, nil
	}
	sys, err := updateSystemPlatform(tx, host.ID, accountID, host, yumUpdates, &updatesReq)
	if err != nil {
		return nil, errors.Wrap(err, "saving system into the database")
	}
	err = tx.Commit().Error
	if err != nil {
		return nil, base.WrapFatalDBError(err, "committing changes")
	}
	return sys, nil
}

func getYumUpdates(event HostEvent, client *api.Client) (*YumUpdates, error) {
	var parsed vmaas.UpdatesV3Response
	res := &YumUpdates{}
	yumUpdates := event.PlatformMetadata.CustomMetadata.YumUpdates
	yumUpdatesURL := event.PlatformMetadata.CustomMetadata.YumUpdatesS3URL

	if yumUpdatesURL != nil && *yumUpdatesURL != "" {
		resp, err := client.Request(&base.Context, http.MethodGet, *yumUpdatesURL, nil, &parsed)
		if err != nil {
			return nil, errors.Wrap(err, "unable to get yum updates from S3")
		}
		if err := resp.Body.Close(); err != nil {
			return nil, errors.Wrap(err, "response error for yum updates from S3")
		}
	}

	if (parsed == vmaas.UpdatesV3Response{}) {
		utils.LogWarn("yum_updates_s3url", yumUpdatesURL, "No yum updates on S3, getting legacy yum_updates field")
		err := json.Unmarshal(yumUpdates, &parsed)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshall yum updates")
		}
	}

	updatesMap := parsed.GetUpdateList()
	if len(updatesMap) == 0 {
		// system does not have any yum updates
		res.RawParsed = yumUpdates
		res.BuiltPkgcache = parsed.GetBuildPkgcache()
		return res, nil
	}
	// we need to get all packages to show up-to-date packages
	installedPkgs := event.Host.SystemProfile.GetInstalledPackages()
	for _, pkg := range installedPkgs {
		if _, has := updatesMap[pkg]; !has {
			updatesMap[pkg] = &vmaas.UpdatesV3ResponseUpdateList{}
		}
	}
	parsed.UpdateList = &updatesMap
	utils.RemoveNonLatestPackages(&parsed)
	yumUpdates, err := json.Marshal(parsed)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshall yum updates")
	}
	res.RawParsed = yumUpdates
	res.BuiltPkgcache = parsed.GetBuildPkgcache()

	utils.LogTrace("inventoryID", event.Host.ID, "yum_updates", string(res.GetRawParsed()))
	return res, nil
}

func (host *Host) GetOrgID() string {
	if host.OrgID == nil {
		return ""
	}
	return *host.OrgID
}
