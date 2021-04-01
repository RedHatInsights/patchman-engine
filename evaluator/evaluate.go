package evaluator

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"context"
	"encoding/json"
	"fmt"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"github.com/jinzhu/gorm"
	"github.com/lestrrat-go/backoff"
	"github.com/pkg/errors"
	"net/http"
	"sync"
	"time"
)

type SystemAdvisoryMap map[string]models.SystemAdvisories

var (
	consumerCount          int
	vmaasClient            *vmaas.APIClient
	evalTopic              string
	evalLabel              string
	port                   string
	enableAdvisoryAnalysis bool
	enablePackageAnalysis  bool
	enableRepoAnalysis     bool
	enableBypass           bool
	enableStaleSysEval     bool
)

func configure() {
	core.ConfigureApp()
	port = utils.GetenvOrFail("PORT")
	traceAPI := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	evalTopic = utils.GetenvOrFail("EVAL_TOPIC")
	evalLabel = utils.GetenvOrFail("EVAL_LABEL")
	consumerCount = utils.GetIntEnvOrFail("CONSUMER_COUNT")

	vmaasConfig := vmaas.NewConfiguration()
	vmaasConfig.BasePath = utils.GetenvOrFail("VMAAS_ADDRESS") + base.VMaaSAPIPrefix
	vmaasConfig.Debug = traceAPI
	disableCompression := !utils.GetBoolEnvOrDefault("ENABLE_VMAAS_CALL_COMPRESSION", true)
	enableAdvisoryAnalysis = utils.GetBoolEnvOrDefault("ENABLE_ADVISORY_ANALYSIS", true)
	enablePackageAnalysis = utils.GetBoolEnvOrDefault("ENABLE_PACKAGE_ANALYSIS", true)
	enableRepoAnalysis = utils.GetBoolEnvOrDefault("ENABLE_REPO_ANALYSIS", true)
	enableStaleSysEval = utils.GetBoolEnvOrDefault("ENABLE_STALE_SYSTEM_EVALUATION", true)
	enableBypass = utils.GetBoolEnvOrDefault("ENABLE_BYPASS", false)
	vmaasConfig.HTTPClient = &http.Client{Transport: &http.Transport{
		DisableCompression: disableCompression,
	}}
	vmaasClient = vmaas.NewAPIClient(vmaasConfig)
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
	return nil
}

func evaluateInDatabase(ctx context.Context, accountID int, inventoryID string,
	requested *base.Rfc3339Timestamp) (*models.SystemPlatform, *vmaas.UpdatesV2Response, error) {
	tx := database.Db.BeginTx(base.Context, nil)
	// Don'requested allow TX to hang around locking the rows
	defer tx.RollbackUnlessCommitted()

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
	vmaasData, err := callVMaas(ctx, updatesReq)
	if err != nil {
		evaluationCnt.WithLabelValues("error-call-vmaas-updates").Inc()
		return nil, errors.Wrap(err, "vmaas API call failed")
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
	if err != nil {
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

	thirdParty, err := analyzeRepos(tx, system)
	if err != nil {
		return errors.Wrap(err, "Repo analysis failed")
	}

	err = updateSystemPlatform(tx, system, newSystemAdvisories, installed, updatable, thirdParty)
	if err != nil {
		evaluationCnt.WithLabelValues("error-update-system").Inc()
		return errors.Wrap(err, "Unable to update system")
	}
	return nil
}

func analyzeRepos(tx *gorm.DB, system *models.SystemPlatform) (
	thirdParty bool, err error) {
	if !enableRepoAnalysis {
		utils.Log().Debug("repo analysis disabled, skipping")
		return false, nil
	}

	// if system has associated at least one third party repo
	// it's marked as third party system
	var thirdPartyCount int
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
	new SystemAdvisoryMap, installed, updatable int, thirdParty bool) error {
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
		counts := make([]int, 4)
		for _, sa := range new {
			if sa.Advisory.AdvisoryTypeID > 0 {
				counts[sa.Advisory.AdvisoryTypeID]++
			}
			counts[0]++
		}
		data["advisory_count_cache"] = counts[0]
		data["advisory_enh_count_cache"] = counts[1]
		data["advisory_bug_count_cache"] = counts[2]
		data["advisory_sec_count_cache"] = counts[3]
	}

	if enablePackageAnalysis {
		data["packages_installed"] = installed
		data["packages_updatable"] = updatable
	}

	if enableRepoAnalysis {
		data["third_party"] = thirdParty
	}

	return tx.Model(system).Update(data).Error
}

// nolint: bodyclose
func callVMaas(ctx context.Context, request *vmaas.UpdatesV3Request) (*vmaas.UpdatesV2Response, error) {
	var policy = backoff.NewExponential(
		backoff.WithInterval(time.Second),
		backoff.WithMaxRetries(8),
	)
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("vmaas-updates-call"))

	vmaasCallArgs := vmaas.AppUpdatesHandlerV3PostPostOpts{
		UpdatesV3Request: optional.NewInterface(*request),
	}
	backoffState, cancel := policy.Start(base.Context)
	defer cancel()
	for backoff.Continue(backoffState) {
		vmaasData, resp, err := vmaasClient.DefaultApi.AppUpdatesHandlerV3PostPost(ctx, &vmaasCallArgs)

		// VMaaS is probably refreshing caches, continue waiting
		if resp != nil && resp.StatusCode == http.StatusServiceUnavailable {
			continue
		}

		if err != nil {
			responseDetails := utils.TryGetResponseDetails(resp)
			return nil, errors.Wrap(err, "vmaas API call failed"+responseDetails+fmt.Sprintf(
				", (packages: %d, basearch: %s, modules: %d, releasever: %s, repolist: %d)",
				len(request.PackageList), request.Basearch, len(request.ModulesList), request.Releasever,
				len(request.RepositoryList)))
		}
		return &vmaasData, nil
	}
	return nil, errors.New("VMaaS is unavailable")
}

func loadSystemData(tx *gorm.DB, accountID int, inventoryID string) (*models.SystemPlatform, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("data-loading"))

	var system models.SystemPlatform
	err := tx.Set("gorm:query_option", "FOR UPDATE OF system_platform").
		Where("rh_account_id = ?", accountID).
		Where("inventory_id = ?::uuid", inventoryID).Find(&system).Error
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

	if event.SystemIDs != nil {
		// Evaluate in bulk
		for _, id := range event.SystemIDs {
			err = Evaluate(base.Context, event.AccountID, id, event.Timestamp, evalLabel)
			if err != nil {
				continue
			}
		}
	} else {
		err = Evaluate(base.Context, event.AccountID, event.ID, event.Timestamp, evalLabel)
	}

	if err != nil {
		utils.Log("err", err.Error(), "inventoryID", event.ID, "evalLabel", evalLabel).
			Error("Eval message handling")
		return err
	}
	utils.Log("inventoryID", event.ID, "evalLabel", evalLabel).Debug("system evaluated successfully")
	return nil
}

func run(wg *sync.WaitGroup, readerBuilder mqueue.CreateReader) {
	utils.Log().Info("evaluator starting")
	configure()

	go RunMetrics(port)

	var handler = mqueue.MakeRetryingHandler(mqueue.MakeMessageHandler(evaluateHandler))
	// We create multiple consumers, and hope that the partition rebalancing
	// algorithm assigns each consumer a single partition
	for i := 0; i < consumerCount; i++ {
		mqueue.SpawnReader(wg, evalTopic, readerBuilder, handler)
	}
}

func RunEvaluator() {
	var wg sync.WaitGroup
	run(&wg, mqueue.ReaderFromEnv)
	wg.Wait()
	utils.Log().Info("evaluator completed")
}
