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

	deleteIDs, installableIDs, applicableIDs, err := processSystemAdvisories(tx, system, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-process-advisories").Inc()
		return nil, errors.Wrap(err, "Unable to process system advisories")
	}

	newSystemAdvisories, err := storeAdvisoryData(tx, system, deleteIDs, installableIDs, applicableIDs)
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
	[]int64, []int64, []int64, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("advisories-processing"))

	reported := getReportedAdvisories(vmaasData)
	oldSystemAdvisories, err := getStoredAdvisoriesMap(tx, system.RhAccountID, system.ID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Unable to get system stored advisories")
	}

	deleteIDs, installableNames, applicableNames := getAdvisoryChanges(reported, oldSystemAdvisories)
	updatesCnt.WithLabelValues("patched").Add(float64(len(deleteIDs)))
	utils.LogInfo("inventoryID", system.InventoryID, "fixed", len(deleteIDs), "fixed advisories")

	installableIDs, err := getAdvisoriesFromDB(tx, installableNames)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Unable to ensure new installable system advisories in db")
	}
	nUnknown := len(installableNames) - len(installableIDs)
	if nUnknown > 0 {
		utils.LogInfo("inventoryID", system.InventoryID, "unknown", nUnknown, "unknown installable advisories - ignored")
		updatesCnt.WithLabelValues("unknown").Add(float64(nUnknown))
	}
	updatesCnt.WithLabelValues("installable").Add(float64(len(installableIDs)))
	utils.LogInfo("inventoryID", system.InventoryID, "installable", len(installableIDs), "installable advisories")

	applicableIDs, err := getAdvisoriesFromDB(tx, applicableNames)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Unable to ensure new applicable system advisories in db")
	}
	nUnknown = len(applicableNames) - len(applicableIDs)
	if nUnknown > 0 {
		utils.LogInfo("inventoryID", system.InventoryID, "unknown", nUnknown, "unknown applicable advisories - ignored")
		updatesCnt.WithLabelValues("unknown").Add(float64(nUnknown))
	}
	updatesCnt.WithLabelValues("applicable").Add(float64(len(applicableIDs)))
	utils.LogInfo("inventoryID", system.InventoryID, "applicable", len(applicableIDs), "applicable advisories")

	return deleteIDs, installableIDs, applicableIDs, nil
}

func storeAdvisoryData(tx *gorm.DB, system *models.SystemPlatform,
	deleteIDs, installableIDs, applicableIDs []int64) (SystemAdvisoryMap, error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("advisories-store"))
	newSystemAdvisories, err := updateSystemAdvisories(tx, system, deleteIDs, installableIDs, applicableIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update system advisories")
	}

	err = updateAdvisoryAccountData(tx, system, deleteIDs, installableIDs, applicableIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update advisory_account_data caches")
	}
	return newSystemAdvisories, nil
}

func getStoredAdvisoriesMap(tx *gorm.DB, accountID int, systemID int64) (map[string]models.SystemAdvisories, error) {
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

func getAdvisoryChanges(reported map[string]int, stored map[string]models.SystemAdvisories) (
	[]int64, []string, []string) {
	installableNames := make([]string, 0, len(reported))
	applicableNames := make([]string, 0, len(reported))
	deleteIDs := make([]int64, 0, len(stored))
	for reportedAdvisory, statusID := range reported {
		if advisory, found := stored[reportedAdvisory]; !found || advisory.StatusID != statusID {
			if statusID == INSTALLABLE {
				installableNames = append(installableNames, reportedAdvisory)
			} else {
				applicableNames = append(applicableNames, reportedAdvisory)
			}
		}
	}
	for storedAdvisory, storedAdvisoryObj := range stored {
		if _, found := reported[storedAdvisory]; !found {
			// advisory was patched from last evaluation,let's remove it
			deleteIDs = append(deleteIDs, storedAdvisoryObj.AdvisoryID)
		}
	}
	return deleteIDs, installableNames, applicableNames
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

func ensureSystemAdvisories(tx *gorm.DB, rhAccountID int, systemID int64, installableIDs,
	applicableIDs []int64) error {
	advisoriesObjs := make(models.SystemAdvisoriesSlice, 0, len(installableIDs)+len(applicableIDs))
	for _, advisoryID := range installableIDs {
		advisoriesObjs = append(advisoriesObjs,
			models.SystemAdvisories{RhAccountID: rhAccountID,
				SystemID:   systemID,
				AdvisoryID: advisoryID,
				StatusID:   INSTALLABLE})
	}
	for _, advisoryID := range applicableIDs {
		advisoriesObjs = append(advisoriesObjs,
			models.SystemAdvisories{RhAccountID: rhAccountID,
				SystemID:   systemID,
				AdvisoryID: advisoryID,
				StatusID:   APPLICABLE})
	}

	tx = database.OnConflictUpdateMulti(tx, []string{"rh_account_id", "system_id", "advisory_id"}, "status_id")
	err := database.BulkInsert(tx, advisoriesObjs)
	return err
}

func lockAdvisoryAccountData(tx *gorm.DB, system *models.SystemPlatform, deleteIDs, installableIDs,
	applicableIDs []int64) error {
	// Lock advisory-account data, so it's not changed by other concurrent queries
	var aads []models.AdvisoryAccountData
	err := tx.Clauses(clause.Locking{
		Strength: "UPDATE",
		Table:    clause.Table{Name: clause.CurrentTable},
	}).Order("advisory_id").
		Find(&aads, "rh_account_id = ? AND (advisory_id in (?) OR advisory_id in (?) OR advisory_id in (?))",
			system.RhAccountID, deleteIDs, installableIDs, applicableIDs).Error

	return err
}

func calcAdvisoryChanges(system *models.SystemPlatform, deleteIDs, installableIDs,
	applicableIDs []int64) []models.AdvisoryAccountData {
	// If system is stale, we won't change any rows  in advisory_account_data
	if system.Stale {
		return []models.AdvisoryAccountData{}
	}

	aadMap := make(map[int64]models.AdvisoryAccountData, len(installableIDs))

	for _, id := range installableIDs {
		aadMap[id] = models.AdvisoryAccountData{
			AdvisoryID:         id,
			RhAccountID:        system.RhAccountID,
			SystemsInstallable: 1,
			// every installable advisory is also applicable advisory
			SystemsApplicable: 1,
		}
	}

	isApplicable := make(map[int64]bool, len(applicableIDs))
	for _, id := range applicableIDs {
		isApplicable[id] = true
		// add advisories which are only applicable and not installable to aad
		if _, ok := aadMap[id]; !ok {
			aadMap[id] = models.AdvisoryAccountData{
				AdvisoryID:        id,
				RhAccountID:       system.RhAccountID,
				SystemsApplicable: 1,
			}
		}
	}

	for _, id := range deleteIDs {
		aadMap[id] = models.AdvisoryAccountData{
			AdvisoryID:         id,
			RhAccountID:        system.RhAccountID,
			SystemsInstallable: -1,
		}
		if !isApplicable[id] {
			// advisory is no longer applicable
			aad := aadMap[id]
			aad.SystemsApplicable = -1
			aadMap[id] = aad
		}
	}

	deltas := make([]models.AdvisoryAccountData, 0, len(deleteIDs)+len(installableIDs)+len(applicableIDs))
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
	deleteIDs, installableIDs, applicableIDs []int64) (SystemAdvisoryMap, error) {
	// this will remove many many old items from our "system_advisories" table
	err := deleteOldSystemAdvisories(tx, system.RhAccountID, system.ID, deleteIDs)
	if err != nil {
		return nil, err
	}

	err = ensureSystemAdvisories(tx, system.RhAccountID, system.ID, installableIDs, applicableIDs)
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

func updateAdvisoryAccountData(tx *gorm.DB, system *models.SystemPlatform, deleteIDs, installableIDs,
	applicableIDs []int64) error {
	err := lockAdvisoryAccountData(tx, system, deleteIDs, installableIDs, applicableIDs)
	if err != nil {
		return err
	}

	changes := calcAdvisoryChanges(system, deleteIDs, installableIDs, applicableIDs)

	if len(changes) == 0 {
		return nil
	}

	txOnConflict := database.OnConflictDoUpdateExpr(tx, []string{"rh_account_id", "advisory_id"},
		database.UpExpr{Name: "systems_installable",
			Expr: "advisory_account_data.systems_installable + excluded.systems_installable"},
		database.UpExpr{Name: "systems_applicable",
			Expr: "advisory_account_data.systems_applicable + excluded.systems_applicable"})

	return database.BulkInsert(txOnConflict, changes)
}
