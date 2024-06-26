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
	"sync"
	"time"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
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
	disableCompression            bool
	enableYumUpdatesEval          bool
	nEvalGoroutines               int
	enableInstantNotifications    bool
	enableSatelliteFunctionality  bool
	errVmaasBadRequest            = errors.New("vmaas bad request")
)

const WarnPayloadTracker = "unable to send message to payload tracker"

func configure() {
	core.ConfigureApp()
	confugureEvaluator()
	evalTopic = utils.FailIfEmpty(utils.CoreCfg.EvalTopic, "EVAL_TOPIC")
	ptTopic = utils.FailIfEmpty(utils.CoreCfg.PayloadTrackerTopic, "PAYLOAD_TRACKER_TOPIC")
	ptWriter = mqueue.NewKafkaWriterFromEnv(ptTopic)
	useTraceLevel := log.IsLevelEnabled(log.TraceLevel)
	vmaasClient = &api.Client{
		HTTPClient: &http.Client{Transport: &http.Transport{DisableCompression: disableCompression}},
		Debug:      useTraceLevel,
	}
	vmaasUpdatesURL = utils.FailIfEmpty(utils.CoreCfg.VmaasAddress, "VMAAS_ADDRESS") + base.VMaaSAPIPrefix + "/updates"
	configureRemediations()
	configureNotifications()
	configureStatus()
}

func confugureEvaluator() {
	evalLabel = utils.FailIfEmpty(utils.PodConfig.GetString("label", ""), "label")
	// Number of kafka readers for upload topic
	consumerCount = utils.PodConfig.GetInt("consumer_count", 1)
	// Toggle compression on vmass API HTTP call
	disableCompression = !utils.PodConfig.GetBool("vmaas_call_compression", true)
	// Evaluate advisories
	enableAdvisoryAnalysis = utils.PodConfig.GetBool("advisory_analysis", true)
	// evaluate packages
	enablePackageAnalysis = utils.PodConfig.GetBool("package_analysis", true)
	// Look for third party repos
	enableRepoAnalysis = utils.PodConfig.GetBool("repo_analysis", true)
	// Evaluate stale systems
	enableStaleSysEval = utils.PodConfig.GetBool("stale_system_evaluation", true)
	// Process (and save to db) previously unknown packages (typically third party packages)
	enableLazyPackageSave = utils.PodConfig.GetBool("lazy_package_save", true)
	// Toggle baseline evaluation
	enableBaselineEval = utils.PodConfig.GetBool("baseline_eval", true)
	// Toggle bypass (fake) messages processing
	enableBypass = utils.PodConfig.GetBool("bypass", false)
	// Toggle in-memory cache to speed up package lookups
	enablePackageCache = utils.PodConfig.GetBool("package_cache", true)
	// Should evaluator load all packages into cache at startup?
	preloadPackageCache = utils.PodConfig.GetBool("package_cache_preload", true)
	// Size of package cache
	packageCacheSize = utils.PodConfig.GetInt("package_cache_size", 1000000)
	// Size of package name cache
	packageNameCacheSize = utils.PodConfig.GetInt("package_name_cache_size", 60000)
	// Toggle caching of vmaas responses
	enableVmaasCache = utils.PodConfig.GetBool("vmaas_cache", true)
	// Size of vmaas response cache
	vmaasCacheSize = utils.PodConfig.GetInt("vmaas_cache_size", 10000)
	// Interval to check vmaas API if there was a data change thus if cache is still valid
	vmaasCacheCheckDuration = time.Duration(utils.PodConfig.GetInt("vmaas_cache_check_duration_sec", 60)) * time.Second
	// How many tries before we consider vmaad API call failed
	vmaasCallMaxRetries = utils.PodConfig.GetInt("vmaas_call_max_retries", 8)
	// Use exponential delay between two vmaas calls
	vmaasCallUseExpRetry = utils.PodConfig.GetBool("vmaas_call_use_exp_retry", true)
	// Set optimistic_update in vmaas API request
	vmaasCallUseOptimisticUpdates = utils.PodConfig.GetBool("vmaas_call_optimistic_updates", true)
	// Include data reported by systems yum/dnf into evaluation
	enableYumUpdatesEval = utils.PodConfig.GetBool("yum_updates_eval", true)
	// How parallel system evaluation we can run
	nEvalGoroutines = utils.PodConfig.GetInt("max_goroutines", 1)
	// Send advisory notification immediately
	enableInstantNotifications = utils.PodConfig.GetBool("instant_notifications", true)
	// Ignore baselines/templates for satellite managed systems
	enableSatelliteFunctionality = utils.PodConfig.GetBool("satellite_functionality", true)
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
	*models.SystemPlatform, *vmaas.UpdatesV3Response, error) {
	system, err := tryGetSystem(event.AccountID, inventoryID, event.Timestamp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to get system")
	}
	if system == nil {
		// warning logged in `tryGetSystem`
		return nil, nil, nil
	}

	thirdParty, err := analyzeRepos(system)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Repo analysis failed")
	}
	system.ThirdParty = thirdParty // to set "system_platform.third_party" column

	updatesData, err := getUpdatesData(ctx, system)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to get updates data")
	}
	if updatesData == nil {
		utils.LogWarn("inventoryID", inventoryID, "No vmaas updates")
		return nil, nil, nil
	}

	// load and evaluate advisories for system
	// posunut spred `updateAdvisoryAccountData`
	// update in `storeAdvisoryData`

	vmaasData, err := evaluateWithVmaas(updatesData, system, event)
	if err != nil {
		return nil, nil, errors.Wrap(err, "evaluation with vmaas failed")
	}

	return system, vmaasData, nil
}

func tryGetYumUpdates(system *models.SystemPlatform) (*vmaas.UpdatesV3Response, error) {
	if system.YumUpdates == nil {
		return nil, nil
	}

	var resp vmaas.UpdatesV3Response
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

	// set EVRA and package name
	for k, v := range updatesMap {
		updates := make([]vmaas.UpdatesV3ResponseAvailableUpdates, 0, len(v.GetAvailableUpdates()))
		for _, u := range v.GetAvailableUpdates() {
			nevra, err := utils.ParseNevra(u.GetPackage())
			if err != nil {
				utils.LogWarn("package", u.GetPackage(), "Cannot parse package")
				continue
			}
			updates = append(updates, vmaas.UpdatesV3ResponseAvailableUpdates{
				Repository:  u.Repository,
				Releasever:  u.Releasever,
				Basearch:    u.Basearch,
				Erratum:     u.Erratum,
				Package:     u.Package,
				PackageName: utils.PtrString(nevra.Name),
				EVRA:        utils.PtrString(nevra.EVRAStringE(true)),
			})
		}
		updatesMap[k] = &vmaas.UpdatesV3ResponseUpdateList{
			AvailableUpdates: &updates,
		}
	}
	return &resp, nil
}

func evaluateWithVmaas(updatesData *vmaas.UpdatesV3Response,
	system *models.SystemPlatform, event *mqueue.PlatformEvent) (*vmaas.UpdatesV3Response, error) {
	if enableBaselineEval {
		if !system.SatelliteManaged || (system.SatelliteManaged && !enableSatelliteFunctionality) {
			err := limitVmaasToBaseline(system, updatesData)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to evaluate baseline")
			}
		}
	}

	err := evaluateAndStore(system, updatesData, event)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to evaluate and store results")
	}
	return updatesData, nil
}

func getUpdatesData(ctx context.Context, system *models.SystemPlatform) (*vmaas.UpdatesV3Response, error) {
	var yumUpdates *vmaas.UpdatesV3Response
	var yumErr error
	if enableYumUpdatesEval {
		yumUpdates, yumErr = tryGetYumUpdates(system)
		if yumErr != nil {
			// ignore broken yum updates
			utils.LogWarn("Can't parse yum_updates", yumErr.Error())
		}
	}

	vmaasData, vmaasErr := getVmaasUpdates(ctx, system)
	if vmaasErr != nil {
		if errors.Is(vmaasErr, errVmaasBadRequest) {
			// vmaas bad request means we either created wrong vmaas request
			// or more likely we received package_list without epochs
			// either way, we should skip this system and not fail hard which will cause pod to restart
			utils.LogWarn("Vmaas response error - bad request, skipping system", vmaasErr.Error())
			return nil, nil
		}
		// if there's no yum update fail hard otherwise only log warning and use yum data
		if yumUpdates == nil {
			return nil, errors.Wrap(vmaasErr, vmaasErr.Error())
		}
		utils.LogWarn("Vmaas response error, continuing with yum updates only", vmaasErr.Error())
	}

	if system.SatelliteManaged {
		// satellite managed systems has vmaas updates APPLICABLE instead of INSTALLABLE
		mergedUpdateList := vmaasData.GetUpdateList()
		for nevra := range mergedUpdateList {
			(*mergedUpdateList[nevra]).SetUpdatesInstallability(APPLICABLE)
		}
	}

	merged := utils.MergeVMaaSResponses(yumUpdates, vmaasData)
	return merged, nil
}

func getVmaasUpdates(ctx context.Context, system *models.SystemPlatform) (*vmaas.UpdatesV3Response, error) {
	var vmaasDataCopy vmaas.UpdatesV3Response
	// first check if we have data in cache
	vmaasData, ok := memoryVmaasCache.Get(system.JSONChecksum)
	if ok {
		// return copy of vmaasData to avoid modification of cached data e.g. by templates
		err := copier.CopyWithOption(&vmaasDataCopy, vmaasData, copier.Option{DeepCopy: true})
		if err != nil {
			return nil, err
		}
		return &vmaasDataCopy, nil
	}
	updatesReq, err := tryGetVmaasRequest(system)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get vmaas request")
	}

	if updatesReq == nil {
		// warning logged in `tryGetVmaasRequest`
		return nil, nil
	}

	updatesReq.ThirdParty = utils.PtrBool(system.ThirdParty) // enable "third_party" updates in VMaaS if needed
	useOptimisticUpdates := system.ThirdParty || vmaasCallUseOptimisticUpdates
	updatesReq.OptimisticUpdates = utils.PtrBool(useOptimisticUpdates)
	updatesReq.EpochRequired = utils.PtrBool(true)

	vmaasData, err = callVMaas(ctx, updatesReq)
	if err != nil {
		evaluationCnt.WithLabelValues("error-call-vmaas-updates").Inc()
		return nil, errors.Wrap(err, "vmaas API call failed")
	}

	if memoryVmaasCache.enabled {
		// store copy of vmaasData to cache to avoid modification of cached data e.g. by templates
		err := copier.CopyWithOption(&vmaasDataCopy, vmaasData, copier.Option{DeepCopy: true})
		if err != nil {
			return nil, err
		}
		memoryVmaasCache.Add(system.JSONChecksum, &vmaasDataCopy)
	}
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

func tryGetSystem(accountID int, inventoryID string,
	requested *types.Rfc3339Timestamp) (*models.SystemPlatform, error) {
	system, err := loadSystemData(accountID, inventoryID)
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

func evaluateAndStore(system *models.SystemPlatform,
	vmaasData *vmaas.UpdatesV3Response, event *mqueue.PlatformEvent) error {
	// deleteIDs, installableIDs, applicableIDs, err := lazySaveAndLoadAdvisories(system, vmaasData)
	// if err != nil {
	// 	return errors.Wrap(err, "Advisory loading failed")
	// }

	advisoriesByName, err := lazySaveAndLoadAdvisories2(system, vmaasData)
	if err != nil {
		return errors.Wrap(err, "Advisory loading failed")
	}

	pkgByName, installed, installable, applicable, err := lazySaveAndLoadPackages(system, vmaasData)
	if err != nil {
		return errors.Wrap(err, "Package loading failed")
	}

	tx := database.DB.WithContext(base.Context).Begin()
	// Don't allow requested TX to hang around locking the rows
	defer tx.Rollback()

	// systemAdvisoriesNew, err := storeAdvisoryData(tx, system, deleteIDs, installableIDs, applicableIDs)
	// if err != nil {
	// 	evaluationCnt.WithLabelValues("error-store-advisories").Inc()
	// 	return errors.Wrap(err, "Unable to store advisory data")
	// }
	systemAdvisoriesNew, err := storeAdvisoryData2(tx, system, advisoriesByName) // TODO: what to do with `systemAdvNew`?
	if err != nil {
		evaluationCnt.WithLabelValues("error-store-advisories").Inc()
		return errors.Wrap(err, "Unable to store advisory data")
	}

	err = updateSystemPackages(tx, system, pkgByName)
	if err != nil {
		evaluationCnt.WithLabelValues("error-system-pkgs").Inc()
		return errors.Wrap(err, "Unable to update system packages")
	}

	err = updateSystemPlatform(tx, system, systemAdvisoriesNew, installed, installable, applicable)
	if err != nil {
		evaluationCnt.WithLabelValues("error-update-system").Inc()
		return errors.Wrap(err, "Unable to update system")
	}

	// Send instant notification with new advisories
	if enableInstantNotifications {
		err = publishNewAdvisoriesNotification(tx, system, event, system.RhAccountID, systemAdvisoriesNew)
		if err != nil {
			evaluationCnt.WithLabelValues("error-advisory-notification").Inc()
			utils.LogError("orgID", event.GetOrgID(), "inventoryID", system.GetInventoryID(), "err", err.Error(),
				"publishing new advisories notification failed")
		}
	}

	err = commitWithObserve(tx)
	if err != nil {
		evaluationCnt.WithLabelValues("error-database-commit").Inc()
		return errors.New("database commit failed")
	}

	return nil
}

func analyzeRepos(system *models.SystemPlatform) (thirdParty bool, err error) {
	if !enableRepoAnalysis {
		utils.LogInfo("repo analysis disabled, skipping")
		return false, nil
	}

	// if system has associated at least one third party repo
	// it's marked as third party system
	var thirdPartyCount int64
	err = database.DB.Table("system_repo sr").
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
	new SystemAdvisoryMap, installed, installable, applicable int) error {
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
		data["packages_installable"] = installable
		data["packages_applicable"] = applicable
	}

	if enableRepoAnalysis {
		data["third_party"] = system.ThirdParty
	}

	if enableSatelliteFunctionality && system.SatelliteManaged && system.BaselineID != nil {
		data["baseline_id"] = nil
		data["baseline_uptodate"] = nil
	}

	err := tx.Model(system).Updates(data).Error

	now := time.Now()
	if system.LastUpload.Sub(now) > time.Hour {
		// log long evaluating systems
		utils.LogWarn("id", system.InventoryID, "lastUpload", *system.LastUpload, "now", now, "uploadEvaluationDelay")
	}
	return err
}

func callVMaas(ctx context.Context, request *vmaas.UpdatesV3Request) (*vmaas.UpdatesV3Response, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("vmaas-updates-call"))

	vmaasCallFunc := func() (interface{}, *http.Response, error) {
		utils.LogTrace("request", *request, "vmaas /updates request")
		vmaasData := vmaas.UpdatesV3Response{}
		resp, err := vmaasClient.Request(&ctx, http.MethodPost, vmaasUpdatesURL, request, &vmaasData)
		statusCode := utils.TryGetStatusCode(resp)
		utils.LogDebug("status_code", statusCode, "vmaas /updates call")
		utils.LogTrace("response", resp, "vmaas /updates response")
		if err != nil && statusCode == 400 {
			err = errors.Wrap(errVmaasBadRequest, err.Error())
		}
		return &vmaasData, resp, err
	}

	vmaasDataPtr, err := utils.HTTPCallRetry(base.Context, vmaasCallFunc, vmaasCallUseExpRetry, vmaasCallMaxRetries,
		http.StatusServiceUnavailable)
	if err != nil {
		return nil, errors.Wrap(err, "vmaas /v3/updates API call failed")
	}
	return vmaasDataPtr.(*vmaas.UpdatesV3Response), nil
}

func loadSystemData(accountID int, inventoryID string) (*models.SystemPlatform, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("data-loading"))

	var system models.SystemPlatform
	err := database.DB.Where("rh_account_id = ?", accountID).
		Where("inventory_id = ?::uuid", inventoryID).
		Find(&system).Error
	return &system, err
}

func parseVmaasJSON(system *models.SystemPlatform) (vmaas.UpdatesV3Request, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("parse-vmaas-json"))
	return utils.ParseVmaasJSON(system)
}

func invalidateCaches(orgID string) error {
	err := database.DB.Model(models.RhAccount{}).
		Where("org_id = ?", orgID).
		Where("valid_package_cache = true OR valid_advisory_cache = true").
		// use map because struct updates only non-zero values and we need to update it to `false`
		Updates(map[string]interface{}{"valid_package_cache": false, "valid_advisory_cache": false}).
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

	if cacheErr := invalidateCaches(event.GetOrgID()); cacheErr != nil {
		utils.LogError("err", err.Error(), "org_id", event.GetOrgID(), "Couldn't invalidate caches")
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
	if memoryVmaasCache.enabled {
		// no need to check cache validity when cache is not enabled
		go memoryVmaasCache.CheckValidity()
	}
}

func run(wg *sync.WaitGroup, readerBuilder mqueue.CreateReader) {
	utils.LogInfo("evaluator starting")
	configure()
	utils.LogDebug("max_goroutines", nEvalGoroutines, "evaluation running in goroutines")

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

	go utils.RunProfiler()

	run(&wg, mqueue.NewKafkaReaderFromEnv)
	wg.Wait()
	utils.LogInfo("evaluator completed")
}
