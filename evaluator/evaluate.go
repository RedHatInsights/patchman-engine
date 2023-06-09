package evaluator

import (
	"app/base"
	"app/base/api"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/types"
	"app/base/utils"
	"app/base/vmaas"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	uploadLabel = "upload"
	recalcLabel = "recalc"
)

type SystemAdvisoryMap map[string]models.SystemAdvisories

var (
	consumerCount                 int
	vmaasClient                   *api.Client
	vmaasUpdatesURL               string
	evalTopic                     string
	evalLabel                     string
	ptTopic                       string
	ptWriter                      mqueue.Writer
	enableAdvisoryAnalysis        bool
	enablePackageAnalysis         bool
	enableRepoAnalysis            bool
	enableBypass                  bool
	enableStaleSysEval            bool
	enableLazyPackageSave         bool
	enableBaselineEval            bool
	prunePackageLatestOnly        bool
	enablePackageCache            bool
	preloadPackageCache           bool
	packageCacheSize              int
	packageNameCacheSize          int
	enableVmaasCache              bool
	vmaasCacheSize                int
	vmaasCacheCheckDuration       time.Duration
	vmaasCallMaxRetries           int
	vmaasCallUseExpRetry          bool
	vmaasCallUseOptimisticUpdates bool
	enableYumUpdatesEval          bool
	nEvalGoroutines               int
	enableInstantNotifications    bool
)

const WarnPayloadTracker = "unable to send message to payload tracker"

func configure() {
	core.ConfigureApp()
	evalTopic = utils.FailIfEmpty(utils.Cfg.EvalTopic, "EVAL_TOPIC")
	evalLabel = utils.GetenvOrFail("EVAL_LABEL")
	ptTopic = utils.FailIfEmpty(utils.Cfg.PayloadTrackerTopic, "PAYLOAD_TRACKER_TOPIC")
	ptWriter = mqueue.NewKafkaWriterFromEnv(ptTopic)
	consumerCount = utils.GetIntEnvOrDefault("CONSUMER_COUNT", 1)
	disableCompression := !utils.GetBoolEnvOrDefault("ENABLE_VMAAS_CALL_COMPRESSION", true)
	enableAdvisoryAnalysis = utils.GetBoolEnvOrDefault("ENABLE_ADVISORY_ANALYSIS", true)
	enablePackageAnalysis = utils.GetBoolEnvOrDefault("ENABLE_PACKAGE_ANALYSIS", true)
	enableRepoAnalysis = utils.GetBoolEnvOrDefault("ENABLE_REPO_ANALYSIS", true)
	enableStaleSysEval = utils.GetBoolEnvOrDefault("ENABLE_STALE_SYSTEM_EVALUATION", true)
	enableLazyPackageSave = utils.GetBoolEnvOrDefault("ENABLE_LAZY_PACKAGE_SAVE", true)
	enableBaselineEval = utils.GetBoolEnvOrDefault("ENABLE_BASELINE_EVAL", true)
	prunePackageLatestOnly = utils.GetBoolEnvOrDefault("PRUNE_UPDATES_LATEST_ONLY", false)
	enableBypass = utils.GetBoolEnvOrDefault("ENABLE_BYPASS", false)
	useTraceLevel := strings.ToLower(utils.Getenv("LOG_LEVEL", "INFO")) == "trace"
	vmaasClient = &api.Client{
		HTTPClient: &http.Client{Transport: &http.Transport{DisableCompression: disableCompression}},
		Debug:      useTraceLevel,
	}
	vmaasUpdatesURL = utils.FailIfEmpty(utils.Cfg.VmaasAddress, "VMAAS_ADDRESS") + base.VMaaSAPIPrefix + "/updates"
	enablePackageCache = utils.GetBoolEnvOrDefault("ENABLE_PACKAGE_CACHE", true)
	preloadPackageCache = utils.GetBoolEnvOrDefault("PRELOAD_PACKAGE_CACHE", true)
	packageCacheSize = utils.GetIntEnvOrDefault("PACKAGE_CACHE_SIZE", 1000000)
	packageNameCacheSize = utils.GetIntEnvOrDefault("PACKAGE_NAME_CACHE_SIZE", 60000)
	enableVmaasCache = utils.GetBoolEnvOrDefault("ENABLE_VMAAS_CACHE", true)
	vmaasCacheSize = utils.GetIntEnvOrDefault("VMAAS_CACHE_SIZE", 10000)
	vmaasCacheCheckDuration = time.Duration(utils.GetIntEnvOrDefault("VMAAS_CACHE_CHECK_DURATION_SEC", 60)) * time.Second
	vmaasCallMaxRetries = utils.GetIntEnvOrDefault("VMAAS_CALL_MAX_RETRIES", 8)
	vmaasCallUseExpRetry = utils.GetBoolEnvOrDefault("VMAAS_CALL_USE_EXP_RETRY", true)
	vmaasCallUseOptimisticUpdates = utils.GetBoolEnvOrDefault("VMAAS_CALL_USE_OPTIMISTIC_UPDATES", true)
	enableYumUpdatesEval = utils.GetBoolEnvOrDefault("ENABLE_YUM_UPDATES_EVAL", true)
	nEvalGoroutines = utils.GetIntEnvOrDefault("MAX_EVAL_GOROUTINES", 1)
	enableInstantNotifications = utils.GetBoolEnvOrDefault("ENABLE_INSTANT_NOTIFICATIONS", true)
	configureRemediations()
	configureNotifications()
	configureStatus()
}

func Evaluate(ctx context.Context, event *mqueue.PlatformEvent, inventoryID, evaluationType string) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationDuration.WithLabelValues(evaluationType))

	utils.LogInfo("inventoryID", inventoryID, "Evaluating system")
	if enableBypass {
		evaluationCnt.WithLabelValues("bypassed").Inc()
		utils.LogInfo("inventoryID", inventoryID, "Evaluation bypassed")
		return nil
	}

	system, vmaasData, err := evaluateInDatabase(ctx, event, inventoryID)
	if err != nil {
		return errors.Wrap(err, "unable to evaluate in database")
	}

	err = publishRemediationsState(system, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-remediations-publish").Inc()
		return errors.Wrap(err, "remediations publish failed")
	}

	if system != nil {
		// increment `success` metric only if the system exists
		// don't count messages from `tryGetSystem` as success
		evaluationCnt.WithLabelValues("success").Inc()
		utils.LogInfo("inventoryID", inventoryID, "evalLabel", evaluationType, "System evaluated successfully")
		return nil
	}
	utils.LogInfo("inventoryID", inventoryID, "evalLabel", evaluationType, "System not evaluated")
	return nil
}

// Runs Evaluate method in Goroutines
func runEvaluate(
	ctx context.Context,
	event mqueue.PlatformEvent, // makes a copy to avoid races
	inventoryID string, // coming from loop
	evaluationType string,
	ptEventIn mqueue.PayloadTrackerEvent,
	wg *sync.WaitGroup,
	guard chan struct{},
) (ptEventOut mqueue.PayloadTrackerEvent, err error) {
	errc := make(chan error, 1)
	ptEventC := make(chan mqueue.PayloadTrackerEvent, 1)

	guard <- struct{}{}
	wg.Add(1)
	ptEventC <- ptEventIn

	go func() {
		err := Evaluate(ctx, &event, inventoryID, evaluationType)
		if err != nil {
			event := <-ptEventC
			event.Status = "error"
			ptEventC <- event
			utils.LogError("err", err.Error(), "inventoryID", inventoryID, "evalLabel", evalLabel,
				"Eval message handling")
		}
		errc <- err
		<-guard
		wg.Done()
	}()

	err = <-errc
	ptEventOut = <-ptEventC
	return ptEventOut, err
}

func evaluateInDatabase(ctx context.Context, event *mqueue.PlatformEvent, inventoryID string) (
	*models.SystemPlatform, *vmaas.UpdatesV2Response, error) {
	tx := database.Db.WithContext(base.Context).Begin()
	// Don't allow requested TX to hang around locking the rows
	defer tx.Rollback()

	system, err := tryGetSystem(tx, event.AccountID, inventoryID, event.Timestamp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to get system")
	}
	if system == nil {
		// warning logged in `tryGetSystem`
		return nil, nil, nil
	}

	updatesData, err := getUpdatesData(ctx, tx, system)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to get updates data")
	}
	if updatesData == nil {
		utils.LogWarn("inventoryID", inventoryID, "No vmaas updates")
		return nil, nil, nil
	}

	vmaasData, err := evaluateWithVmaas(tx, updatesData, system, event)
	if err != nil {
		return nil, nil, errors.Wrap(err, "evaluation with vmaas failed")
	}

	return system, vmaasData, nil
}

func tryGetYumUpdates(system *models.SystemPlatform) (*vmaas.UpdatesV2Response, error) {
	if system.YumUpdates == nil {
		return nil, nil
	}

	var resp vmaas.UpdatesV2Response
	err := json.Unmarshal(system.YumUpdates, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshall yum updates")
	}
	updatesMap := resp.GetUpdateList()
	if len(updatesMap) == 0 {
		// TODO: do we need evaluationCnt.WithLabelValues("error-no-yum-packages").Inc()?
		utils.LogWarn("inventoryID", system.GetInventoryID(), "No yum_updates")
		return nil, nil
	}

	return &resp, nil
}

func evaluateWithVmaas(tx *gorm.DB, updatesData *vmaas.UpdatesV2Response,
	system *models.SystemPlatform, event *mqueue.PlatformEvent) (*vmaas.UpdatesV2Response, error) {
	if enableBaselineEval {
		err := limitVmaasToBaseline(tx, system, updatesData)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to evaluate baseline")
		}
	}

	err := evaluateAndStore(tx, system, updatesData, event)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to evaluate and store results")
	}

	err = commitWithObserve(tx)
	if err != nil {
		evaluationCnt.WithLabelValues("error-database-commit").Inc()
		return nil, errors.New("database commit failed")
	}
	return updatesData, nil
}

func getUpdatesData(ctx context.Context, tx *gorm.DB, system *models.SystemPlatform) (
	*vmaas.UpdatesV2Response, error) {
	var yumUpdates *vmaas.UpdatesV2Response
	var yumErr error
	if enableYumUpdatesEval {
		yumUpdates, yumErr = tryGetYumUpdates(system)
		if yumErr != nil {
			// ignore broken yum updates
			utils.LogWarn("Can't parse yum_updates", yumErr.Error())
		}
	}

	vmaasData, vmaasErr := getVmaasUpdates(ctx, tx, system)
	if vmaasErr != nil {
		// if there's no yum update fail hard otherwise only log warning and use yum data
		if yumUpdates == nil {
			return nil, errors.Wrap(vmaasErr, vmaasErr.Error())
		}
		utils.LogWarn("Vmaas response error, continuing with yum updates only", vmaasErr.Error())
	}

	// Try to merge YumUpdates and VMaaS updates
	updatesData, err := utils.MergeVMaaSResponses(vmaasData, yumUpdates)
	if err != nil {
		return nil, err
	}

	return updatesData, nil
}

func getVmaasUpdates(ctx context.Context, tx *gorm.DB,
	system *models.SystemPlatform) (*vmaas.UpdatesV2Response, error) {
	// first check if we have data in cache
	vmaasData, ok := memoryVmaasCache.Get(system.JSONChecksum)
	if ok {
		return vmaasData, nil
	}
	updatesReq, err := tryGetVmaasRequest(system)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get vmaas request")
	}

	if updatesReq == nil {
		// warning logged in `tryGetVmaasRequest`
		return nil, nil
	}

	thirdParty, err := analyzeRepos(tx, system)
	if err != nil {
		return nil, errors.Wrap(err, "Repo analysis failed")
	}
	system.ThirdParty = thirdParty                    // to set "system_platform.third_party" column
	updatesReq.ThirdParty = utils.PtrBool(thirdParty) // enable "third_party" updates in VMaaS if needed
	useOptimisticUpdates := thirdParty || vmaasCallUseOptimisticUpdates
	updatesReq.OptimisticUpdates = utils.PtrBool(useOptimisticUpdates)
	updatesReq.EpochRequired = utils.PtrBool(true)

	vmaasData, err = callVMaas(ctx, updatesReq)
	if err != nil {
		evaluationCnt.WithLabelValues("error-call-vmaas-updates").Inc()
		return nil, errors.Wrap(err, "vmaas API call failed")
	}

	memoryVmaasCache.Add(system.JSONChecksum, vmaasData)
	return vmaasData, nil
}

func tryGetVmaasRequest(system *models.SystemPlatform) (*vmaas.UpdatesV3Request, error) {
	if system == nil || system.VmaasJSON == nil {
		evaluationCnt.WithLabelValues("error-parse-vmaas-json").Inc()
		utils.LogWarn("inventoryID", system.GetInventoryID(), "system with empty vmaas json")
		// skip the system
		// don't return error as it will cause panic of evaluator pod
		return nil, nil
	}

	updatesReq, err := parseVmaasJSON(system)
	if err != nil {
		evaluationCnt.WithLabelValues("error-parse-vmaas-json").Inc()
		return nil, errors.Wrap(err, "Unable to parse system vmaas json")
	}

	if len(updatesReq.PackageList) == 0 {
		evaluationCnt.WithLabelValues("error-no-packages").Inc()
		utils.LogWarn("inventoryID", system.GetInventoryID(), "Empty package list")
		return nil, nil
	}

	if len(updatesReq.RepositoryList) == 0 {
		// system without any repositories won't have any advisories evaluated by vmaas
		evaluationCnt.WithLabelValues("error-no-repositories").Inc()
		utils.LogWarn("inventoryID", system.GetInventoryID(), "Empty repository list")
		return nil, nil
	}
	return &updatesReq, nil
}

func tryGetSystem(tx *gorm.DB, accountID int, inventoryID string,
	requested *types.Rfc3339Timestamp) (*models.SystemPlatform, error) {
	system, err := loadSystemData(tx, accountID, inventoryID)
	if err != nil {
		evaluationCnt.WithLabelValues("error-db-read-inventory-data").Inc()
		return nil, errors.Wrap(err, "error loading system from DB")
	}
	if system.ID == 0 {
		evaluationCnt.WithLabelValues("error-db-read-inventory-data").Inc()
		utils.LogWarn("inventoryID", inventoryID, "System not found in DB")
		return nil, nil
	}

	if system.Stale && !enableStaleSysEval {
		evaluationCnt.WithLabelValues("skipping-stale").Inc()
		utils.LogWarn("inventoryID", inventoryID, "Skipping stale system")
		return nil, nil
	}

	if requested != nil && system.LastEvaluation != nil && requested.Time().Before(*system.LastEvaluation) {
		evaluationCnt.WithLabelValues("skip-old-msg").Inc()
		utils.LogWarn("inventoryID", inventoryID, "Skipping old message")
		return nil, nil
	}
	return system, nil
}

func commitWithObserve(tx *gorm.DB) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("commit-to-db"))

	err := tx.Commit().Error
	if err != nil {
		return err
	}
	return nil
}

func evaluateAndStore(tx *gorm.DB, system *models.SystemPlatform,
	vmaasData *vmaas.UpdatesV2Response, event *mqueue.PlatformEvent) error {
	newSystemAdvisories, err := analyzeAdvisories(tx, system, vmaasData)
	if err != nil {
		return errors.Wrap(err, "Advisory analysis failed")
	}

	installed, updatable, err := analyzePackages(tx, system, vmaasData)
	if err != nil {
		return errors.Wrap(err, "Package analysis failed")
	}

	err = updateSystemPlatform(tx, system, newSystemAdvisories, installed, updatable)
	if err != nil {
		evaluationCnt.WithLabelValues("error-update-system").Inc()
		return errors.Wrap(err, "Unable to update system")
	}

	// Send instant notification with new advisories
	if enableInstantNotifications {
		err = publishNewAdvisoriesNotification(tx, system, event, system.RhAccountID, newSystemAdvisories)
		if err != nil {
			evaluationCnt.WithLabelValues("error-advisory-notification").Inc()
			utils.LogError("orgID", event.GetOrgID(), "inventoryID", system.GetInventoryID(), "err", err.Error(),
				"publishing new advisories notification failed")
		}
	}

	return nil
}

func analyzeRepos(tx *gorm.DB, system *models.SystemPlatform) (
	thirdParty bool, err error) {
	if !enableRepoAnalysis {
		utils.LogInfo("repo analysis disabled, skipping")
		return false, nil
	}

	// if system has associated at least one third party repo
	// it's marked as third party system
	var thirdPartyCount int64
	err = tx.Table("system_repo sr").
		Joins("join repo r on r.id = sr.repo_id").
		Where("sr.rh_account_id = ?", system.RhAccountID).
		Where("sr.system_id = ?", system.ID).
		Where("r.third_party = true").
		Count(&thirdPartyCount).Error
	if err != nil {
		utils.LogWarn("err", err.Error(), "accountID", system.RhAccountID, "systemID", system.ID,
			"counting third party repos")
		return false, err
	}
	thirdParty = thirdPartyCount > 0
	return thirdParty, nil
}

// nolint: funlen
func updateSystemPlatform(tx *gorm.DB, system *models.SystemPlatform,
	new SystemAdvisoryMap, installed, updatable int) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("system-update"))
	defer utils.ObserveSecondsSince(*system.LastUpload, uploadEvaluationDelay)
	if system.LastEvaluation != nil {
		defer utils.ObserveHoursSince(*system.LastEvaluation, twoEvaluationsInterval)
	}

	data := make(map[string]interface{}, 8)
	data["last_evaluation"] = time.Now()

	if enableAdvisoryAnalysis {
		if new == nil {
			return errors.New("Invalid args")
		}
		installableCount := 0
		installableEnhCount := 0
		installableBugCount := 0
		installableSecCount := 0
		applicableCount := 0
		applicableEnhCount := 0
		applicableBugCount := 0
		applicableSecCount := 0
		for _, sa := range new {
			if sa.StatusID == INSTALLABLE {
				switch sa.Advisory.AdvisoryTypeID {
				case 1:
					installableEnhCount++
				case 2:
					installableBugCount++
				case 3:
					installableSecCount++
				}
				installableCount++
			}
			switch sa.Advisory.AdvisoryTypeID {
			case 1:
				applicableEnhCount++
			case 2:
				applicableBugCount++
			case 3:
				applicableSecCount++
			}
			applicableCount++
		}

		data["installable_advisory_count_cache"] = installableCount
		data["installable_advisory_enh_count_cache"] = installableEnhCount
		data["installable_advisory_bug_count_cache"] = installableBugCount
		data["installable_advisory_sec_count_cache"] = installableSecCount

		data["applicable_advisory_count_cache"] = applicableCount
		data["applicable_advisory_enh_count_cache"] = applicableEnhCount
		data["applicable_advisory_bug_count_cache"] = applicableBugCount
		data["applicable_advisory_sec_count_cache"] = applicableSecCount
	}

	if enablePackageAnalysis {
		data["packages_installed"] = installed
		data["packages_updatable"] = updatable
	}

	if enableRepoAnalysis {
		data["third_party"] = system.ThirdParty
	}

	return tx.Model(system).Updates(data).Error
}

func callVMaas(ctx context.Context, request *vmaas.UpdatesV3Request) (*vmaas.UpdatesV2Response, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("vmaas-updates-call"))

	vmaasCallFunc := func() (interface{}, *http.Response, error) {
		utils.LogTrace("request", *request, "vmaas /updates request")
		vmaasData := vmaas.UpdatesV2Response{}
		resp, err := vmaasClient.Request(&ctx, http.MethodPost, vmaasUpdatesURL, request, &vmaasData)
		utils.LogDebug("status_code", utils.TryGetStatusCode(resp), "vmaas /updates call")
		utils.LogTrace("response", resp, "vmaas /updates response")
		return &vmaasData, resp, err
	}

	vmaasDataPtr, err := utils.HTTPCallRetry(base.Context, vmaasCallFunc, vmaasCallUseExpRetry, vmaasCallMaxRetries,
		http.StatusServiceUnavailable)
	if err != nil {
		return nil, errors.Wrap(err, "vmaas /v3/updates API call failed")
	}
	return vmaasDataPtr.(*vmaas.UpdatesV2Response), nil
}

func loadSystemData(tx *gorm.DB, accountID int, inventoryID string) (*models.SystemPlatform, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("data-loading"))

	var system models.SystemPlatform
	err := tx.Clauses(clause.Locking{
		Strength: "UPDATE",
		Table:    clause.Table{Name: clause.CurrentTable},
	}).Where("rh_account_id = ?", accountID).
		Where("inventory_id = ?::uuid", inventoryID).
		Find(&system).Error
	return &system, err
}

func parseVmaasJSON(system *models.SystemPlatform) (vmaas.UpdatesV3Request, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("parse-vmaas-json"))

	var updatesReq vmaas.UpdatesV3Request
	err := json.Unmarshal([]byte(*system.VmaasJSON), &updatesReq)
	return updatesReq, err
}

func invalidatePkgCache(orgID string) error {
	err := database.Db.Model(models.RhAccount{}).
		Where("org_id = ?", orgID).
		Update("valid_package_cache", false).
		Error
	return err
}

func evaluateHandler(event mqueue.PlatformEvent) error {
	var err error
	var wg sync.WaitGroup
	guard := make(chan struct{}, nEvalGoroutines)

	nSystems := 1
	if event.SystemIDs != nil {
		nSystems = len(event.SystemIDs)
	}
	ptEvents := make(mqueue.PayloadTrackerEvents, 0, nSystems)
	ptEvent := mqueue.PayloadTrackerEvent{
		OrgID:     event.OrgID,
		Status:    "success",
		StatusMsg: "advisories evaluation",
	}

	if event.SystemIDs != nil {
		// Evaluate in bulk
		nRequestIDs := len(event.RequestIDs)
		for i, id := range event.SystemIDs {
			ptEvent.InventoryID = id
			if nRequestIDs > i {
				ptEvent.RequestID = &event.RequestIDs[i]
			}
			ptEvent, err = runEvaluate(base.Context, event, id, evalLabel, ptEvent, &wg, guard)
			ptEvents = append(ptEvents, ptEvent)
		}
	} else {
		ptEvent, err = runEvaluate(base.Context, event, event.ID, evalLabel, ptEvent, &wg, guard)
		ptEvents = append(ptEvents, ptEvent)
	}
	wg.Wait()

	if cacheErr := invalidatePkgCache(event.GetOrgID()); cacheErr != nil {
		utils.LogError("err", err.Error(), "org_id", event.GetOrgID(), "Couldn't invalidate pkg cache")
	}

	// send kafka message to payload tracker
	if evalLabel == uploadLabel {
		ptErr := mqueue.SendMessages(base.Context, ptWriter, &ptEvents)
		if ptErr != nil {
			// don't fail with err, just log that we couldn't send msg to payload tracker
			utils.LogWarn("err", ptErr.Error(), WarnPayloadTracker)
		}
	}
	return err
}

func loadCache() {
	memoryPackageCache = NewPackageCache(enablePackageCache, preloadPackageCache, packageCacheSize, packageNameCacheSize)
	memoryPackageCache.Load()
	memoryVmaasCache = NewVmaasPackageCache(enableVmaasCache, vmaasCacheSize, vmaasCacheCheckDuration)
	go memoryVmaasCache.CheckValidity()
}

func run(wg *sync.WaitGroup, readerBuilder mqueue.CreateReader) {
	utils.LogInfo("evaluator starting")
	configure()
	utils.LogDebug("MAX_EVAL_GOROUTINES", nEvalGoroutines, "evaluation running in goroutines")

	go RunMetrics()

	loadCache()

	var handler = mqueue.MakeRetryingHandler(mqueue.MakeMessageHandler(evaluateHandler))
	// We create multiple consumers, and hope that the partition rebalancing
	// algorithm assigns each consumer a single partition
	for i := 0; i < consumerCount; i++ {
		mqueue.SpawnReader(wg, evalTopic, readerBuilder, handler)
	}
}

func RunEvaluator() {
	var wg sync.WaitGroup
	run(&wg, mqueue.NewKafkaReaderFromEnv)
	wg.Wait()
	utils.LogInfo("evaluator completed")
}
