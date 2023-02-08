package evaluator

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func analyzeAdvisories(tx *gorm.DB, system *models.SystemPlatform, vmaasData *vmaas.UpdatesV2Response) (
	SystemAdvisoryMap, error) {
	if !enableAdvisoryAnalysis {
		utils.LogInfo("advisory analysis disabled, skipping")
		return nil, nil
	}

	deleteAIDs, addAIDs, err := processSystemAdvisories(tx, system, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-process-advisories").Inc()
		return nil, errors.Wrap(err, "Unable to process system advisories")
	}

	newSystemAdvisories, err := storeAdvisoryData(tx, system, deleteAIDs, addAIDs)
	if err != nil {
		evaluationCnt.WithLabelValues("error-store-advisories").Inc()
		return nil, errors.Wrap(err, "Unable to store advisory data")
	}
	return newSystemAdvisories, nil
}

// Changes data stored in system_advisories, in order to match newest evaluation
// Before this methods stores the entries into the system_advisories table, it locks
// advisory_account_data table, so other evaluations don't interfere with this one
func processSystemAdvisories(tx *gorm.DB, system *models.SystemPlatform, vmaasData *vmaas.UpdatesV2Response) (
	deleteAIDs []int64, addAIDs []int64, err error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("advisories-processing"))

	reported := getReportedAdvisories(vmaasData)
	oldSystemAdvisories, err := getStoredAdvisoriesMap(tx, system.RhAccountID, int(system.ID))
	if err != nil {
		return nil, nil, errors.Wrap(err, "Unable to get system stored advisories")
	}

	deleteAIDs, newAdvisoryNames := getAdvisoryChanges(reported, oldSystemAdvisories)
	updatesCnt.WithLabelValues("patched").Add(float64(len(deleteAIDs)))
	utils.LogInfo("inventoryID", system.InventoryID, "patched", len(deleteAIDs), "patched advisories")
	utils.LogInfo("inventoryID", system.InventoryID, "newAdvisories", len(newAdvisoryNames), "new advisories")

	addAIDs, err = getAdvisoriesFromDB(tx, newAdvisoryNames)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Unable to ensure new system advisories in db")
	}
	nUnknown := len(newAdvisoryNames) - len(addAIDs)
	if nUnknown > 0 {
		utils.LogInfo("inventoryID", system.InventoryID, "unknown", nUnknown, "unknown advisories - ignored")
		updatesCnt.WithLabelValues("unknown").Add(float64(nUnknown))
	}

	updatesCnt.WithLabelValues("unpatched").Add(float64(len(addAIDs)))
	utils.LogInfo("inventoryID", system.InventoryID, "unpatched", len(addAIDs), "patched advisories")
	return deleteAIDs, addAIDs, nil
}

func storeAdvisoryData(tx *gorm.DB, system *models.SystemPlatform,
	deleteAIDs, addAIDs []int64) (SystemAdvisoryMap, error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("advisories-store"))
	newSystemAdvisories, err := updateSystemAdvisories(tx, system, deleteAIDs, addAIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update system advisories")
	}

	err = updateAdvisoryAccountData(tx, system, deleteAIDs, addAIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update advisory_account_data caches")
	}
	return newSystemAdvisories, nil
}

func getStoredAdvisoriesMap(tx *gorm.DB, accountID, systemID int) (map[string]models.SystemAdvisories, error) {
	var advisories []models.SystemAdvisories
	err := database.SystemAdvisoriesBySystemID(tx, accountID, systemID).Preload("Advisory").Find(&advisories).Error
	if err != nil {
		return nil, err
	}

	advisoriesMap := make(map[string]models.SystemAdvisories, len(advisories))
	for _, advisory := range advisories {
		advisoriesMap[advisory.Advisory.Name] = advisory
	}
	return advisoriesMap, nil
}

func getAdvisoryChanges(reported map[string]bool, stored map[string]models.SystemAdvisories) (
	[]int64, []string) {
	newAdvisoryNames := make([]string, 0, len(reported))
	deleteAIDs := make([]int64, 0, len(stored))
	for reportedAdvisory := range reported {
		if _, found := stored[reportedAdvisory]; !found {
			newAdvisoryNames = append(newAdvisoryNames, reportedAdvisory)
		}
	}
	for storedAdvisory, storedAdvisoryObj := range stored {
		if _, found := reported[storedAdvisory]; !found {
			// advisory was patched from last evaluation,let's remove it
			deleteAIDs = append(deleteAIDs, storedAdvisoryObj.AdvisoryID)
		}
	}
	return deleteAIDs, newAdvisoryNames
}

// Return advisory IDs, created advisories count, error
func getAdvisoriesFromDB(tx *gorm.DB, advisories []string) ([]int64, error) {
	var advisoryIDs []int64
	err := tx.Model(&models.AdvisoryMetadata{}).Where("name IN (?)", advisories).
		Pluck("id", &advisoryIDs).Error
	if err != nil {
		return nil, err
	}
	return advisoryIDs, nil
}

func ensureSystemAdvisories(tx *gorm.DB, rhAccountID int, systemID int64, advisoryIDs []int64) error {
	advisoriesObjs := models.SystemAdvisoriesSlice{}
	for _, advisoryID := range advisoryIDs {
		advisoriesObjs = append(advisoriesObjs,
			models.SystemAdvisories{RhAccountID: rhAccountID, SystemID: systemID, AdvisoryID: advisoryID})
	}

	txOnConflict := tx.Clauses(clause.OnConflict{
		DoNothing: true,
	})
	err := database.BulkInsert(txOnConflict, advisoriesObjs)
	return err
}

func lockAdvisoryAccountData(tx *gorm.DB, system *models.SystemPlatform, deleteAIDs, addAIDs []int64) error {
	// Lock advisory-account data, so it's not changed by other concurrent queries
	var aads []models.AdvisoryAccountData
	err := tx.Clauses(clause.Locking{
		Strength: "UPDATE",
		Table:    clause.Table{Name: clause.CurrentTable},
	}).Order("advisory_id").
		Find(&aads, "rh_account_id = ? AND (advisory_id in (?) OR advisory_id in (?))",
			system.RhAccountID, deleteAIDs, addAIDs).Error

	return err
}

func calcAdvisoryChanges(system *models.SystemPlatform, deleteAIDs, addAIDs []int64) []models.AdvisoryAccountData {
	// If system is stale, we won't change any rows  in advisory_account_data
	if system.Stale {
		return []models.AdvisoryAccountData{}
	}
	aadMap := make(map[int64]models.AdvisoryAccountData, len(addAIDs))

	for _, id := range addAIDs {
		aadMap[id] = models.AdvisoryAccountData{
			AdvisoryID:      id,
			RhAccountID:     system.RhAccountID,
			SystemsAffected: 1,
		}
	}

	for _, id := range deleteAIDs {
		aadMap[id] = models.AdvisoryAccountData{
			AdvisoryID:      id,
			RhAccountID:     system.RhAccountID,
			SystemsAffected: -1,
		}
	}

	deltas := make([]models.AdvisoryAccountData, 0, len(deleteAIDs)+len(addAIDs))
	for _, aad := range aadMap {
		deltas = append(deltas, aad)
	}
	return deltas
}

func deleteOldSystemAdvisories(tx *gorm.DB, accountID int, systemID int64, patched []int64) error {
	err := tx.Where("rh_account_id = ? ", accountID).
		Where("system_id = ?", systemID).
		Where("advisory_id in (?)", patched).
		Delete(&models.SystemAdvisories{}).Error
	return err
}

func updateSystemAdvisories(tx *gorm.DB, system *models.SystemPlatform,
	deleteAIDs, addAIDs []int64) (SystemAdvisoryMap, error) {
	// this will remove many many old items from our "system_advisories" table
	err := deleteOldSystemAdvisories(tx, system.RhAccountID, system.ID, deleteAIDs)
	if err != nil {
		return nil, err
	}

	err = ensureSystemAdvisories(tx, system.RhAccountID, system.ID, addAIDs)
	if err != nil {
		return nil, err
	}

	newSystemAdvisories := SystemAdvisoryMap{}
	var data []models.SystemAdvisories
	err = tx.Preload("Advisory").
		Find(&data, "system_id = ? AND rh_account_id = ?", system.ID, system.RhAccountID).Error

	if err != nil {
		return nil, err
	}

	for _, sa := range data {
		newSystemAdvisories[sa.Advisory.Name] = sa
	}
	return newSystemAdvisories, nil
}

func updateAdvisoryAccountData(tx *gorm.DB, system *models.SystemPlatform, deleteAIDs, addAIDs []int64) error {
	err := lockAdvisoryAccountData(tx, system, deleteAIDs, addAIDs)
	if err != nil {
		return err
	}

	changes := calcAdvisoryChanges(system, deleteAIDs, addAIDs)

	if len(changes) == 0 {
		return nil
	}

	txOnConflict := database.OnConflictDoUpdateExpr(tx, []string{"rh_account_id", "advisory_id"},
		database.UpExpr{Name: "systems_affected", Expr: "advisory_account_data.systems_affected + excluded.systems_affected"})

	return database.BulkInsert(txOnConflict, changes)
}
