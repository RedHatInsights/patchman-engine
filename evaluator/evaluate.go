package evaluator

import (
	"app/base"
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
	"github.com/pkg/errors"
	"time"
)

const (
	unknown = "unknown"
)

var (
	consumerCount int
	vmaasClient   *vmaas.APIClient
	evalTopic     string
	evalLabel     string
	port          string
)

func configure() {
	port = utils.GetenvOrFail("PORT")
	traceAPI := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	evalTopic = utils.GetenvOrFail("EVAL_TOPIC")
	evalLabel = utils.GetenvOrFail("EVAL_LABEL")
	consumerCount = utils.GetIntEnvOrFail("CONSUMER_COUNT")

	vmaasConfig := vmaas.NewConfiguration()
	vmaasConfig.BasePath = utils.GetenvOrFail("VMAAS_ADDRESS") + base.VMaaSAPIPrefix
	vmaasConfig.Debug = traceAPI
	vmaasClient = vmaas.NewAPIClient(vmaasConfig)
}

func Evaluate(ctx context.Context, inventoryID string, evaluationType string) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationDuration.WithLabelValues(evaluationType))

	tx := database.Db.Begin()
	// Don't allow TX to hang around locking the rows
	defer tx.RollbackUnlessCommitted()

	system, err := loadSystemData(tx, inventoryID)
	if err != nil {
		evaluationCnt.WithLabelValues("error-db-read-inventory-data").Inc()
		return errors.Wrap(err, "Unable to get system data from database")
	}

	updatesReq, err := parseVmaasJSON(system)
	if err != nil {
		evaluationCnt.WithLabelValues("error-parse-vmaas-json").Inc()
		return errors.Wrap(err, "Unable to parse system vmaas json")
	}

	if len(updatesReq.PackageList) == 0 {
		evaluationCnt.WithLabelValues("error-no-packages").Inc()
		return errors.New("No packages found in vmaas_json")
	}

	vmaasData, err := callVMaas(ctx, updatesReq)
	if err != nil {
		evaluationCnt.WithLabelValues("error-call-vmaas-updates").Inc()
		return errors.New("vmaas API call failed")
	}

	err = evaluateAndStore(tx, system, *vmaasData)
	if err != nil {
		return errors.Wrap(err, "Unable to evaluate and store results")
	}

	err = commitWithObserve(tx)
	if err != nil {
		evaluationCnt.WithLabelValues("error-database-commit").Inc()
		return errors.New("database commit failed")
	}

	evaluationCnt.WithLabelValues("success").Inc()
	return nil
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

func evaluateAndStore(tx *gorm.DB, system *models.SystemPlatform, vmaasData vmaas.UpdatesV2Response) error {
	patched, unpatched, err := processSystemAdvisories(tx, system, vmaasData, system.InventoryID)
	if err != nil {
		evaluationCnt.WithLabelValues("error-process-advisories").Inc()
		return errors.Wrap(err, "Unable to process system advisories")
	}

	err = storeAdvisoryData(tx, system, patched, unpatched)
	if err != nil {
		evaluationCnt.WithLabelValues("error-store-advisories").Inc()
		return errors.Wrap(err, "Unable to store advisory data")
	}

	err = updateCaches(tx, system)
	if err != nil {
		evaluationCnt.WithLabelValues("error-update-system-caches").Inc()
		return errors.Wrap(err, "Unable to update system caches")
	}

	err = updateSystemLastEvaluation(tx, system)
	if err != nil {
		evaluationCnt.WithLabelValues("error-update-last-eval").Inc()
		return errors.Wrap(err, "Unable to update last_evaluation timestamp")
	}
	return nil
}

func updateSystemLastEvaluation(tx *gorm.DB, system *models.SystemPlatform) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("last-evaluation-update"))

	err := tx.Model(&models.SystemPlatform{}).Where("id = ?", system.ID).
		Update("last_evaluation", time.Now()).Error
	return err
}

func updateCaches(tx *gorm.DB, system *models.SystemPlatform) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("caches-update"))

	err := tx.Exec("SELECT * FROM refresh_system_caches(?,?)", system.ID, system.RhAccountID).Error

	return err
}

func callVMaas(ctx context.Context, updatesReq vmaas.UpdatesV3Request) (*vmaas.UpdatesV2Response, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("vmaas-updates-call"))

	vmaasCallArgs := vmaas.AppUpdatesHandlerV3PostPostOpts{
		UpdatesV3Request: optional.NewInterface(updatesReq),
	}

	vmaasData, resp, err := vmaasClient.UpdatesApi.AppUpdatesHandlerV3PostPost(ctx, &vmaasCallArgs)
	if err != nil {
		responseDetails := utils.TryGetResponseDetails(resp)
		return nil, errors.Wrap(err, "vmaas API call failed"+responseDetails+fmt.Sprintf(
			", (packages: %d, basearch: %s, modules: %d, releasever: %s, repolist: %d, seconly: %t)",
			len(updatesReq.PackageList), updatesReq.Basearch, len(updatesReq.ModulesList), updatesReq.Releasever,
			len(updatesReq.RepositoryList), updatesReq.SecurityOnly))
	}

	return &vmaasData, nil
}

func loadSystemData(tx *gorm.DB, inventoryID string) (*models.SystemPlatform, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("data-loading"))

	var system models.SystemPlatform
	err := tx.Set("gorm:query_option", "FOR UPDATE OF system_platform").
		Where("inventory_id = ?", inventoryID).Find(&system).Error
	return &system, err
}

func parseVmaasJSON(system *models.SystemPlatform) (vmaas.UpdatesV3Request, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("parse-vmaas-json"))

	var updatesReq vmaas.UpdatesV3Request
	err := json.Unmarshal([]byte(system.VmaasJSON), &updatesReq)
	return updatesReq, err
}

// Changes data stored in system_advisories, in order to match newest evaluation
// Before this methods stores the entries into the system_advisories table, it locks
// advisory_account_data table, so other evaluations don't interfere with this one
func processSystemAdvisories(tx *gorm.DB, system *models.SystemPlatform, vmaasData vmaas.UpdatesV2Response,
	inventoryID string) (patched []int, unpatched []int, err error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("advisories-processing"))

	reported := getReportedAdvisories(vmaasData)
	stored, err := getStoredAdvisoriesMap(tx, system.ID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Unable to get system stored advisories")
	}

	patched = getPatchedAdvisories(reported, stored)
	updatesCnt.WithLabelValues("patched").Add(float64(len(patched)))
	utils.Log("inventoryID", inventoryID, "patched", len(patched)).Debug("patched advisories")

	newsAdvisoriesNames, unpatched := getNewAndUnpatchedAdvisories(reported, stored)
	utils.Log("inventoryID", inventoryID, "newAdvisories", len(newsAdvisoriesNames)).Debug("new advisories")

	newIDs, err := ensureAdvisoriesInDb(tx, newsAdvisoriesNames)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Unable to ensure new system advisories in db")
	}

	unpatched = append(unpatched, newIDs...)
	updatesCnt.WithLabelValues("unpatched").Add(float64(len(unpatched)))
	utils.Log("inventoryID", inventoryID, "unpatched", len(unpatched)).Debug("patched advisories")
	return patched, unpatched, nil
}

func storeAdvisoryData(tx *gorm.DB, system *models.SystemPlatform, patched, unpatched []int) error {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("advisories-store"))
	err := updateSystemAdvisories(tx, system, patched, unpatched)
	if err != nil {
		return errors.Wrap(err, "Unable to update system advisories")
	}

	err = updateAdvisoryAccountDatas(tx, system, patched, unpatched)
	if err != nil {
		return errors.Wrap(err, "Unable to update advisory_account_data caches")
	}
	return nil
}

func getReportedAdvisories(vmaasData vmaas.UpdatesV2Response) map[string]bool {
	advisories := map[string]bool{}
	for _, updates := range vmaasData.UpdateList {
		for _, update := range updates.AvailableUpdates {
			advisories[update.Erratum] = true
		}
	}
	return advisories
}

func getStoredAdvisoriesMap(tx *gorm.DB, systemID int) (map[string]models.SystemAdvisories, error) {
	var advisories []models.SystemAdvisories
	err := database.SystemAdvisoriesQueryByID(tx, systemID).Preload("Advisory").Find(&advisories).Error
	if err != nil {
		return nil, err
	}

	advisoriesMap := map[string]models.SystemAdvisories{}
	for _, advisory := range advisories {
		advisoriesMap[advisory.Advisory.Name] = advisory
	}
	return advisoriesMap, nil
}

func getNewAndUnpatchedAdvisories(reported map[string]bool, stored map[string]models.SystemAdvisories) (
	[]string, []int) {
	newAdvisories := []string{}
	unpatchedAdvisories := []int{}
	for reportedAdvisory := range reported {
		if storedAdvisory, found := stored[reportedAdvisory]; found {
			if storedAdvisory.WhenPatched != nil { // this advisory was already patched and now is un-patched again
				unpatchedAdvisories = append(unpatchedAdvisories, storedAdvisory.AdvisoryID)
			}
		} else {
			newAdvisories = append(newAdvisories, reportedAdvisory)
		}
	}
	return newAdvisories, unpatchedAdvisories
}

func getPatchedAdvisories(reported map[string]bool, stored map[string]models.SystemAdvisories) []int {
	patchedAdvisories := make([]int, 0, len(stored))
	for storedAdvisory, storedAdvisoryObj := range stored {
		if _, found := reported[storedAdvisory]; found {
			continue
		}

		// advisory contained in reported - it's patched
		if storedAdvisoryObj.WhenPatched != nil {
			// it's already marked as patched
			continue
		}

		// advisory was patched from last evaluation, let's mark it as patched
		patchedAdvisories = append(patchedAdvisories, storedAdvisoryObj.AdvisoryID)
	}
	return patchedAdvisories
}

func updateSystemAdvisoriesWhenPatched(tx *gorm.DB, system *models.SystemPlatform, advisoryIDs []int,
	whenPatched *time.Time) error {
	err := tx.Model(models.SystemAdvisories{}).
		Where("system_id = ? AND advisory_id IN (?)", system.ID, advisoryIDs).
		Update("when_patched", whenPatched).Error
	if err != nil {
		return err
	}
	return nil
}

// Return advisory IDs, created advisories count, error
func ensureAdvisoriesInDb(tx *gorm.DB, advisories []string) ([]int, error) {
	advisoryObjs := make(models.AdvisoryMetadataSlice, len(advisories))
	for i, advisory := range advisories {
		advisoryObjs[i] = models.AdvisoryMetadata{Name: advisory,
			Description: unknown, Synopsis: unknown, Summary: unknown, Solution: unknown}
	}

	txOnConflict := tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	err := database.BulkInsert(txOnConflict, advisoryObjs)
	if err != nil {
		return nil, err
	}

	var advisoryIDs []int
	err = tx.Model(&models.AdvisoryMetadata{}).Where("name IN (?)", advisories).
		Pluck("id", &advisoryIDs).Error
	if err != nil {
		return nil, err
	}
	return advisoryIDs, nil
}

func ensureSystemAdvisories(tx *gorm.DB, systemID int, advisoryIDs []int) error {
	advisoriesObjs := models.SystemAdvisoriesSlice{}
	for _, advisoryID := range advisoryIDs {
		advisoriesObjs = append(advisoriesObjs,
			models.SystemAdvisories{SystemID: systemID, AdvisoryID: advisoryID})
	}

	txOnConflict := tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	err := database.BulkInsert(txOnConflict, advisoriesObjs)
	return err
}

func lockAdvisoryAccountData(tx *gorm.DB, system *models.SystemPlatform, patched, unpatched []int) error {
	// Lock advisory-account data, so it's not changed by other concurrent queries
	var aads []models.AdvisoryAccountData
	err := tx.Set("gorm:query_option", "FOR UPDATE OF advisory_account_data").
		Order("advisory_id").
		Find(&aads, "rh_account_id = ? AND (advisory_id in (?) OR advisory_id in (?))",
			system.RhAccountID, patched, unpatched).Error

	return err
}

func calcAdvisoryChanges(system *models.SystemPlatform, patched, unpatched []int) []models.AdvisoryAccountData {
	aadMap := map[int]models.AdvisoryAccountData{}
	// If system is stale, we won't change any rows  in advisory_account_data
	if system.Stale {
		return []models.AdvisoryAccountData{}
	}

	for _, id := range unpatched {
		aadMap[id] = models.AdvisoryAccountData{
			AdvisoryID:      id,
			RhAccountID:     system.RhAccountID,
			SystemsAffected: 1,
		}
	}

	for _, id := range patched {
		aadMap[id] = models.AdvisoryAccountData{
			AdvisoryID:      id,
			RhAccountID:     system.RhAccountID,
			SystemsAffected: -1,
		}
	}

	deltas := make([]models.AdvisoryAccountData, 0, len(patched)+len(unpatched))
	for _, aad := range aadMap {
		deltas = append(deltas, aad)
	}
	return deltas
}

func updateSystemAdvisories(tx *gorm.DB, system *models.SystemPlatform, patched, unpatched []int) error {
	whenPatched := time.Now()

	err := ensureSystemAdvisories(tx, system.ID, unpatched)
	if err != nil {
		return err
	}

	err = updateSystemAdvisoriesWhenPatched(tx, system, patched, &whenPatched)
	if err != nil {
		return err
	}

	return updateSystemAdvisoriesWhenPatched(tx, system, unpatched, nil)
}

func updateAdvisoryAccountDatas(tx *gorm.DB, system *models.SystemPlatform, patched, unpatched []int) error {
	err := lockAdvisoryAccountData(tx, system, patched, unpatched)
	if err != nil {
		return err
	}

	changes := calcAdvisoryChanges(system, patched, unpatched)
	txOnConflict := database.OnConflictDoUpdateExpr(tx, []string{"rh_account_id", "advisory_id"},
		database.UpExpr{Name: "systems_affected", Expr: "advisory_account_data.systems_affected + excluded.systems_affected"})

	return database.BulkInsert(txOnConflict, changes)
}

func evaluateHandler(event mqueue.PlatformEvent) error {
	err := Evaluate(context.Background(), event.ID, evalLabel)
	if err != nil {
		utils.Log("err", err.Error(), "inventoryID", event.ID, "evalLabel", evalLabel).
			Error("Eval message handling")
		return err
	}
	utils.Log("inventoryID", event.ID, "evalLabel", evalLabel).Debug("system evaluated successfully")
	return nil
}

func run(readerBuilder mqueue.CreateReader) {
	utils.Log().Info("evaluator starting")
	configure()

	go RunMetrics(port)

	var handler = mqueue.MakeRetryingHandler(mqueue.MakeMessageHandler(evaluateHandler))
	// We create multiple consumers, and hope that the partition rebalancing
	// algorithm assigns each consumer a single partition
	for i := 0; i < consumerCount; i++ {
		go mqueue.RunReader(evalTopic, readerBuilder, handler)
	}
}

func RunEvaluator() {
	run(mqueue.ReaderFromEnv)
	<-make(chan bool)
}
