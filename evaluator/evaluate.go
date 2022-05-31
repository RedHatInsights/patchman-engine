package evaluator

import (
	"app/base"
	"app/base/api"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
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
	vmaasCallMaxRetries           int
	vmaasCallUseExpRetry          bool
	vmaasCallUseOptimisticUpdates bool
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
	vmaasCallMaxRetries = utils.GetIntEnvOrDefault("VMAAS_CALL_MAX_RETRIES", 8)
	vmaasCallUseExpRetry = utils.GetBoolEnvOrDefault("VMAAS_CALL_USE_EXP_RETRY", true)
	vmaasCallUseOptimisticUpdates = utils.GetBoolEnvOrDefault("VMAAS_CALL_USE_OPTIMISTIC_UPDATES", true)
	configureRemediations()
	configureNotifications()
}

func Evaluate(ctx context.Context, accountID int, inventoryID string, requested *base.Rfc3339Timestamp,
	evaluationType string) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationDuration.WithLabelValues(evaluationType))
	if enableBypass {
		evaluationCnt.WithLabelValues("bypassed").Inc()
		utils.Log("inventoryID", inventoryID).Info("Evaluation bypassed")
		return nil
	}

	system, vmaasData, err := evaluateInDatabase(ctx, accountID, inventoryID, requested)
	if err != nil {
		return errors.Wrap(err, "unable to evaluate in database")
	}

	err = publishRemediationsState(system, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-remediations-publish").Inc()
		return errors.Wrap(err, "remediations publish failed")
	}

	evaluationCnt.WithLabelValues("success").Inc()
	utils.Log("inventoryID", inventoryID, "evalLabel", evaluationType).Info("system evaluated successfully")
	return nil
}

func evaluateInDatabase(ctx context.Context, accountID int, inventoryID string,
	requested *base.Rfc3339Timestamp) (*models.SystemPlatform, *vmaas.UpdatesV2Response, error) {
	tx := database.Db.WithContext(base.Context).Begin()
	// Don't allow requested TX to hang around locking the rows
	defer tx.Rollback()

	updatesReq, system, err := tryGetVmaasRequest(tx, accountID, inventoryID, requested)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to get vmaas request")
	}

	if updatesReq == nil {
		return nil, nil, nil
	}

	vmaasData, err := evaluateWithVmaas(ctx, tx, updatesReq, system)
	if err != nil {
		return nil, nil, errors.Wrap(err, "evaluation with vmaas failed")
	}
	return system, vmaasData, nil
}

func evaluateWithVmaas(ctx context.Context, tx *gorm.DB, updatesReq *vmaas.UpdatesV3Request,
	system *models.SystemPlatform) (*vmaas.UpdatesV2Response, error) {
	thirdParty, err := analyzeRepos(tx, system)
	if err != nil {
		return nil, errors.Wrap(err, "Repo analysis failed")
	}
	system.ThirdParty = thirdParty                    // to set "system_platform.third_party" column
	updatesReq.ThirdParty = utils.PtrBool(thirdParty) // enable "third_party" updates in VMaaS if needed
	useOptimisticUpdates := thirdParty || vmaasCallUseOptimisticUpdates
	updatesReq.OptimisticUpdates = utils.PtrBool(useOptimisticUpdates)

	vmaasData, err := callVMaas(ctx, updatesReq)
	if err != nil {
		evaluationCnt.WithLabelValues("error-call-vmaas-updates").Inc()
		return nil, errors.Wrap(err, "vmaas API call failed")
	}

	if enableBaselineEval {
		err = limitVmaasToBaseline(tx, system, vmaasData)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to evaluate baseline")
		}
	}

	err = evaluateAndStore(tx, system, vmaasData)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to evaluate and store results")
	}

	err = commitWithObserve(tx)
	if err != nil {
		evaluationCnt.WithLabelValues("error-database-commit").Inc()
		return nil, errors.New("database commit failed")
	}
	return vmaasData, nil
}

func tryGetVmaasRequest(tx *gorm.DB, accountID int, inventoryID string,
	requested *base.Rfc3339Timestamp) (*vmaas.UpdatesV3Request, *models.SystemPlatform, error) {
	system := tryGetSystem(tx, accountID, inventoryID, requested)
	if system == nil {
		return nil, nil, nil
	}

	updatesReq, err := parseVmaasJSON(system)
	if err != nil {
		evaluationCnt.WithLabelValues("error-parse-vmaas-json").Inc()
		return nil, nil, errors.Wrap(err, "Unable to parse system vmaas json")
	}

	if len(updatesReq.PackageList) == 0 {
		evaluationCnt.WithLabelValues("error-no-packages").Inc()
		return nil, nil, nil
	}
	return &updatesReq, system, nil
}

func tryGetSystem(tx *gorm.DB, accountID int, inventoryID string,
	requested *base.Rfc3339Timestamp) *models.SystemPlatform {
	system, err := loadSystemData(tx, accountID, inventoryID)
	if err != nil || system.ID == 0 {
		evaluationCnt.WithLabelValues("error-db-read-inventory-data").Inc()
		return nil
	}

	if system.Stale && !enableStaleSysEval {
		evaluationCnt.WithLabelValues("skipping-stale").Inc()
		return nil
	}

	if requested != nil && system.LastEvaluation != nil && requested.Time().Before(*system.LastEvaluation) {
		evaluationCnt.WithLabelValues("error-old-msg").Inc()
		return nil
	}
	return system
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

func evaluateAndStore(tx *gorm.DB, system *models.SystemPlatform, vmaasData *vmaas.UpdatesV2Response) error {
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
	err = publishNewAdvisoriesNotification(system.InventoryID, system.RhAccountID, newSystemAdvisories)
	if err != nil {
		utils.Log("RHAccountID", system.RhAccountID, "newSystemAdvisories", newSystemAdvisories).
			Error("publishing new advisories notification failed")
	}

	return nil
}

func analyzeRepos(tx *gorm.DB, system *models.SystemPlatform) (
	thirdParty bool, err error) {
	if !enableRepoAnalysis {
		utils.Log().Info("repo analysis disabled, skipping")
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
		utils.Log("err", err.Error(), "accountID", system.RhAccountID, "systemID", system.ID).
			Warn("counting third party repos")
		return false, err
	}
	thirdParty = thirdPartyCount > 0
	return thirdParty, nil
}

func updateSystemPlatform(tx *gorm.DB, system *models.SystemPlatform,
	new SystemAdvisoryMap, installed, updatable int) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("system-update"))
	defer utils.ObserveSecondsSince(*system.LastUpload, uploadEvaluationDelay)
	if system.LastEvaluation != nil {
		defer utils.ObserveHoursSince(*system.LastEvaluation, twoEvaluationsInterval)
	}

	data := map[string]interface{}{}
	data["last_evaluation"] = time.Now()

	if enableAdvisoryAnalysis {
		if new == nil {
			return errors.New("Invalid args")
		}
		count := 0
		enhCount := 0
		bugCount := 0
		secCount := 0
		for _, sa := range new {
			switch sa.Advisory.AdvisoryTypeID {
			case 1:
				enhCount++
			case 2:
				bugCount++
			case 3:
				secCount++
			}
			count++
		}
		data["advisory_count_cache"] = count
		data["advisory_enh_count_cache"] = enhCount
		data["advisory_bug_count_cache"] = bugCount
		data["advisory_sec_count_cache"] = secCount
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
		utils.Log("request", *request).Trace("vmaas /updates request")
		vmaasData := vmaas.UpdatesV2Response{}
		resp, err := vmaasClient.Request(&ctx, http.MethodPost, vmaasUpdatesURL, request, &vmaasData)
		utils.Log("status_code", utils.TryGetStatusCode(resp)).Debug("vmaas /updates call")
		utils.Log("response", resp).Trace("vmaas /updates response")
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
	err := json.Unmarshal([]byte(system.VmaasJSON), &updatesReq)
	return updatesReq, err
}

func evaluateHandler(event mqueue.PlatformEvent) error {
	var err error
	nSystems := 1
	if event.SystemIDs != nil {
		nSystems = len(event.SystemIDs)
	}
	ptEvents := make(mqueue.PayloadTrackerEvents, 0, nSystems)
	ptEvent := mqueue.PayloadTrackerEvent{
		Account:   event.Account,
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
			err = Evaluate(base.Context, event.AccountID, id, event.Timestamp, evalLabel)
			if err != nil {
				ptEvent.Status = "error"
				ptEvents = append(ptEvents, ptEvent)
				continue
			}
			ptEvents = append(ptEvents, ptEvent)
		}
	} else {
		err = Evaluate(base.Context, event.AccountID, event.ID, event.Timestamp, evalLabel)
	}

	if err != nil {
		utils.Log("err", err.Error(), "inventoryID", event.ID, "evalLabel", evalLabel).
			Error("Eval message handling")
		if event.SystemIDs == nil {
			// this is an evaluator-recalc event, we don't need to send info to payload tracker
			return err
		}
		// send kafka message to payload tracker
		ptErr := mqueue.SendMessages(base.Context, ptWriter, &ptEvents)
		if ptErr != nil {
			utils.Log("err", err.Error()).Warn(WarnPayloadTracker)
		}
		return err
	}

	// send kafka message to payload tracker
	err = mqueue.SendMessages(base.Context, ptWriter, &ptEvents)
	if err != nil {
		// don't fail with err, just log that we couldn't send msg to payload tracker
		utils.Log("err", err.Error()).Warn(WarnPayloadTracker)
	}
	return nil
}

func loadCache() {
	memoryPackageCache = NewPackageCache(enablePackageCache, preloadPackageCache, packageCacheSize, packageNameCacheSize)
	memoryPackageCache.Load()
}

func run(wg *sync.WaitGroup, readerBuilder mqueue.CreateReader) {
	utils.Log().Info("evaluator starting")
	configure()

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
	utils.Log().Info("evaluator completed")
}
