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
	kafkaReader *mqueue.Reader
	vmaasClient *vmaas.APIClient
	evalLabel   string
	port        string
)

func Configure() {
	port = utils.GetenvOrFail("PORT")
	traceAPI := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	evalTopic := utils.GetenvOrFail("EVAL_TOPIC")
	evalLabel = utils.GetenvOrFail("EVAL_LABEL")

	kafkaReader = mqueue.ReaderFromEnv(evalTopic)

	vmaasConfig := vmaas.NewConfiguration()
	vmaasConfig.BasePath = utils.GetenvOrFail("VMAAS_ADDRESS") + base.VMaaSAPIPrefix
	vmaasConfig.Debug = traceAPI
	vmaasClient = vmaas.NewAPIClient(vmaasConfig)
}

func Evaluate(ctx context.Context, inventoryID string, evaluationType string) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationDuration.WithLabelValues(evaluationType))

	system, err := loadSystemData(inventoryID)
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
		return errors.Wrap(err, "Unable to get updates from VMaaS "+fmt.Sprintf(
			"(packages: %d, basearch: %s, modules: %d, releasever: %s, repolist: %d, seconly: %t)",
			len(updatesReq.PackageList), updatesReq.Basearch, len(updatesReq.ModulesList), updatesReq.Releasever,
			len(updatesReq.RepositoryList), updatesReq.SecurityOnly))
	}

	tx := database.Db.Begin()
	err = evaluateAndStore(tx, system, vmaasData)
	if err != nil {
		rollbackOrLogError(tx, inventoryID)
		return errors.Wrap(err, "Unable to evaluate and store results")
	}

	commitOrLogError(tx, inventoryID)

	evaluationCnt.WithLabelValues("success").Inc()
	return nil
}

func commitOrLogError(tx *gorm.DB, inventoryID string) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("commit-to-db"))

	err := tx.Commit().Error
	if err != nil {
		utils.Log("err", err.Error(), "systemID", inventoryID).Error("Unable to commit tx")
	}
}

func rollbackOrLogError(tx *gorm.DB, inventoryID string) {
	err := tx.Rollback().Error
	if err != nil {
		utils.Log("err", err.Error(), "systemID", inventoryID).Error("Unable to rollback tx")
	}
}

func evaluateAndStore(tx *gorm.DB, system *models.SystemPlatform, vmaasData vmaas.UpdatesV2Response) error {
	err := processSystemAdvisories(tx, system.ID, system.RhAccountID, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-process-advisories").Inc()
		return errors.Wrap(err, "Unable to process system advisories")
	}

	err = updateSystemCaches(tx, system)
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

func updateSystemCaches(tx *gorm.DB, system *models.SystemPlatform) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("caches-update"))

	err := tx.Exec("SELECT * FROM update_system_caches(?)", system.ID).Error
	return err
}

func callVMaas(ctx context.Context, updatesReq vmaas.UpdatesV3Request) (vmaas.UpdatesV2Response, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("vmaas-updates-call"))

	vmaasCallArgs := vmaas.AppUpdatesHandlerV3PostPostOpts{
		UpdatesV3Request: optional.NewInterface(updatesReq),
	}

	vmaasData, _, err := vmaasClient.UpdatesApi.AppUpdatesHandlerV3PostPost(ctx, &vmaasCallArgs)
	return vmaasData, err
}

func loadSystemData(inventoryID string) (*models.SystemPlatform, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("data-loading"))

	var system models.SystemPlatform
	err := database.Db.Where("inventory_id = ?", inventoryID).Find(&system).Error
	return &system, err
}

func parseVmaasJSON(system *models.SystemPlatform) (vmaas.UpdatesV3Request, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("parse-vmaas-json"))

	var updatesReq vmaas.UpdatesV3Request
	err := json.Unmarshal([]byte(system.VmaasJSON), &updatesReq)
	return updatesReq, err
}

func processSystemAdvisories(tx *gorm.DB, systemID, rhAccountID int, vmaasData vmaas.UpdatesV2Response) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("advisories-processing"))

	reported := getReportedAdvisories(vmaasData)
	stored, err := getStoredAdvisoriesMap(tx, systemID)
	if err != nil {
		return errors.Wrap(err, "Unable to get system stored advisories")
	}

	patched := getPatchedAdvisories(reported, *stored)
	updatesCnt.WithLabelValues("patched").Add(float64(len(patched)))
	utils.Log("systemID", systemID, "patched", len(patched)).Debug("patched advisories")

	newsAdvisoriesNames, unpatched := getNewAndUnpatchedAdvisories(reported, *stored)
	utils.Log("systemID", systemID, "newAdvisories", len(newsAdvisoriesNames)).Debug("new advisories")

	newIDs, err := ensureAdvisoriesInDb(tx, newsAdvisoriesNames)
	if err != nil {
		return errors.Wrap(err, "Unable to ensure new system advisories in db")
	}

	unpatched = append(unpatched, *newIDs...)
	updatesCnt.WithLabelValues("unpatched").Add(float64(len(unpatched)))
	utils.Log("systemID", systemID, "unpatched", len(unpatched)).Debug("patched advisories")

	err = updateSystemAdvisories(tx, systemID, rhAccountID, patched, unpatched)
	if err != nil {
		return errors.Wrap(err, "Unable to update system advisories")
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

func getStoredAdvisoriesMap(tx *gorm.DB, systemID int) (*map[string]models.SystemAdvisories, error) {
	var advisories []models.SystemAdvisories
	err := database.SystemAdvisoriesQueryByID(tx, systemID).Preload("Advisory").Find(&advisories).Error
	if err != nil {
		return nil, err
	}

	advisoriesMap := map[string]models.SystemAdvisories{}
	for _, advisory := range advisories {
		advisoriesMap[advisory.Advisory.Name] = advisory
	}
	return &advisoriesMap, nil
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
			utils.Log("advisory", storedAdvisory.Advisory.Name).Debug("still not patched")
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

func updateSystemAdvisoriesWhenPatched(tx *gorm.DB, systemID, rhAccountID int, advisoryIDs []int,
	whenPatched *time.Time) error {
	err := tx.Model(models.SystemAdvisories{}).
		Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
		Update("when_patched", whenPatched).Error
	if err != nil {
		return err
	}

	affectedSystemIncrement := 0
	if whenPatched != nil {
		affectedSystemIncrement = -1
	} else {
		affectedSystemIncrement = 1
	}

	err = updateAccountAdvisoriesAffectedSystems(tx, rhAccountID, advisoryIDs, affectedSystemIncrement)
	return err
}

func updateAccountAdvisoriesAffectedSystems(tx *gorm.DB, rhAccountID int, advisoryIDs []int,
	affectedSystemIncrement int) error {
	err := tx.Model(models.AdvisoryAccountData{}).
		Where("rh_account_id = ? AND advisory_id IN (?)", rhAccountID, advisoryIDs).
		UpdateColumn("systems_affected", gorm.Expr("systems_affected + ?", affectedSystemIncrement)).Error
	return err
}

// Return advisory IDs, created advisories count, error
func ensureAdvisoriesInDb(tx *gorm.DB, advisories []string) (*[]int, error) {
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
	return &advisoryIDs, nil
}

func ensureSystemAdvisories(tx *gorm.DB, systemID int, advisoryIDs []int) error {
	advisoriesObjs := models.SystemAdvisoriesSlice{}
	for _, advisoryID := range advisoryIDs {
		advisoriesObjs = append(advisoriesObjs,
			models.SystemAdvisories{SystemID: systemID, AdvisoryID: advisoryID})
	}

	interfaceSlice := advisoriesObjs
	txOnConflict := tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	err := database.BulkInsert(txOnConflict, interfaceSlice)
	return err
}

func ensureAdvisoryAccountDataInDb(tx *gorm.DB, rhAccountID int, advisoryIDs []int) error {
	accountData := make(models.AdvisoryAccountDataSlice, len(advisoryIDs))
	for i, advisoryID := range advisoryIDs {
		accountData[i] = models.AdvisoryAccountData{RhAccountID: rhAccountID, AdvisoryID: advisoryID}
	}

	txOnConflict := tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	err := database.BulkInsert(txOnConflict, accountData)
	return err
}

func updateSystemAdvisories(tx *gorm.DB, systemID, rhAccountID int, patched, unpatched []int) error {
	whenPatched := time.Now()
	err := updateSystemAdvisoriesWhenPatched(tx, systemID, rhAccountID, patched, &whenPatched)
	if err != nil {
		return err
	}

	// delete items with no system related
	err = tx.Where("rh_account_id = ? AND systems_affected = 0", rhAccountID).
		Delete(&models.AdvisoryAccountData{}).Error
	if err != nil {
		return err
	}

	err = ensureSystemAdvisories(tx, systemID, unpatched)
	if err != nil {
		return err
	}

	err = ensureAdvisoryAccountDataInDb(tx, rhAccountID, unpatched)
	if err != nil {
		return err
	}

	err = updateSystemAdvisoriesWhenPatched(tx, systemID, rhAccountID, unpatched, nil)
	return err
}

func RunEvaluator() {
	Configure()

	go RunMetrics(port)

	kafkaReader.HandleEvents(func(event mqueue.PlatformEvent) {
		err := Evaluate(context.Background(), event.ID, evalLabel)
		if err != nil {
			utils.Log("err", err.Error(), "inventoryID", event.ID, "evalLabel", evalLabel).
				Error("Eval message handling")
		}
		utils.Log("inventoryID", event.ID, "evalLabel", evalLabel).Debug("system evaluated successfully")
	})
}
