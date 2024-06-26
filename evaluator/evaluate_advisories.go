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

func lazySaveAndLoadAdvisories(system *models.SystemPlatform, vmaasData *vmaas.UpdatesV3Response) (
	[]int64, []int64, []int64, error) {
	if !enableAdvisoryAnalysis {
		utils.LogInfo("advisory analysis disabled, skipping lazy saving and loading")
		return nil, nil, nil, nil
	}
	deleteIDs, installableIDs, applicableIDs, err := processSystemAdvisories(system, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-process-advisories").Inc()
		return nil, nil, nil, errors.Wrap(err, "unable to process system advisories")
	}

	return deleteIDs, installableIDs, applicableIDs, err
}

func lazySaveAndLoadAdvisories2(system *models.SystemPlatform, vmaasData *vmaas.UpdatesV3Response) (
	ExtendedAdvisoryMap, error) {
	if !enableAdvisoryAnalysis {
		utils.LogInfo("advisory analysis disabled, skipping lazy saving and loading")
		return nil, nil
	}

	// TODO: should this first evaluate missing and lazy-save just the missing?
	err := lazySaveAdvisories2(vmaasData, system.InventoryID)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to store unknown advisories in DB")
	}

	stored, err := loadSystemAdvisories(database.Db, system.RhAccountID, system.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to load system advisories")
	}

	merged, err := evaluateChanges(vmaasData, stored, system.RhAccountID, system.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to evaluate advisory changes")
	}

	return merged, nil
}

func evaluateChanges(vmaasData *vmaas.UpdatesV3Response, stored SystemAdvisoryMap, accountID int, systemID int64) (
	ExtendedAdvisoryMap, error) {
	reported := getReportedAdvisories(vmaasData)

	// TODO: check this func for errors and efficiency

	// -> map[string]int: "erratum -> helper column to diferentiate installable/applicable"
	// --> moze sa porovnavat if statusID == INSTALLABLE {...} else {APPLICABLE...}

	extendedAdvisories := make(ExtendedAdvisoryMap, len(reported)+len(stored))
	missingNames := make([]string, 0, len(reported))
	for reportedName, reportedStatusID := range reported {
		if storedAdvisory, found := stored[reportedName]; found {
			if reportedStatusID != storedAdvisory.StatusID {
				extendedAdvisories[reportedName] = ExtendedAdvisory{
					Change:           Update,
					SystemAdvisories: storedAdvisory,
				}
			} else {
				extendedAdvisories[reportedName] = ExtendedAdvisory{
					Change:           Keep,
					SystemAdvisories: storedAdvisory,
				}
			}
		} else {
			extendedAdvisories[reportedName] = ExtendedAdvisory{
				Change: Add,
				SystemAdvisories: models.SystemAdvisories{
					RhAccountID: accountID,
					SystemID:    systemID,
					StatusID:    reportedStatusID,
				},
			}
			missingNames = append(missingNames, reportedName)
		}
	}

	advisoryMetadata := make(models.AdvisoryMetadataSlice, 0, len(missingNames))
	err := database.Db.Model(&models.AdvisoryMetadata{}).
		Where("name IN (?)", missingNames).
		Select("id, name").
		Scan(&advisoryMetadata).Error
	if err != nil {
		return nil, err
	}

	name2AdvisoryID := make(map[string]int64, len(missingNames))
	for _, am := range advisoryMetadata {
		name2AdvisoryID[am.Name] = am.ID
	}

	for _, name := range missingNames {
		if _, found := name2AdvisoryID[name]; !found {
			return nil, errors.New("Failed to evaluate changes because an advisory was not lazy-saved")
		}
		extendedAdvisory := extendedAdvisories[name]
		extendedAdvisory.AdvisoryID = name2AdvisoryID[name]
		extendedAdvisories[name] = extendedAdvisory
	}

	for storedName, storedAdvisory := range stored {
		if _, found := reported[storedName]; !found {
			extendedAdvisories[storedName] = ExtendedAdvisory{
				Change:           Remove,
				SystemAdvisories: storedAdvisory,
			}
		}
	}

	return extendedAdvisories, nil
}

func lazySaveAdvisories2(vmaasData *vmaas.UpdatesV3Response, inventoryID string) error {
	// -> load reported advisories, advisories to lazy-save can only appear in VmaasData
	reportedNames := getReportedAdvisoryNames(vmaasData)
	if len(reportedNames) < 1 {
		return nil
	}
	// -> get missing from reported
	missingNames, err := getMissingAdvisories(reportedNames) // namiesto getAdvisoriesFromDB
	if err != nil {
		return errors.Wrap(err, "Unable to get missing system advisories")
	}
	// -> log missing found
	nUnknown := len(missingNames)
	if nUnknown > 0 {
		utils.LogInfo("inventoryID", inventoryID, "unknown", nUnknown, "unknown advisories")
		updatesCnt.WithLabelValues("unknown").Add(float64(nUnknown))
	} else {
		return nil
	}
	// -> store missing advisories
	err = storeMissingAdvisories2(missingNames)
	if err != nil {
		return errors.Wrap(err, "failed to save missing advisory_metadata")
	}

	return nil
}

func storeMissingAdvisories2(missingNames []string) error {
	toStore := make(models.AdvisoryMetadataSlice, 0, len(missingNames))
	for _, name := range missingNames {
		if len(name) > 0 && len(name) < 100 {
			toStore = append(toStore, models.AdvisoryMetadata{
				Name:           name,
				Description:    "Not Available",
				Synopsis:       "Not Available",
				Summary:        "Not Available",
				AdvisoryTypeID: 0,
				RebootRequired: false,
				Synced:         false,
			})
		}
	}

	var err error
	if len(toStore) > 0 {
		tx := database.DB.Begin()
		defer tx.Commit()
		err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&toStore).Error
		if err != nil {
			return err
		}
		// TODO: created toStore objects will have hadded a .ID
	}
	return nil
}

// Determine if advisories from DB are propperly stored based on advisory metadata existance.
func getMissingAdvisories(advisoryNames []string) ([]string, error) {
	advisoryMetadata := make(models.AdvisoryMetadataSlice, 0, len(advisoryNames))
	err := database.DB.Model(&models.AdvisoryMetadata{}).
		Where("name IN (?)", advisoryNames).
		Select("id, name").
		Scan(&advisoryMetadata).Error
	if err != nil {
		return nil, err
	}

	found := make(map[string]bool, len(advisoryNames))
	for _, am := range advisoryMetadata {
		found[am.Name] = true
	}

	missingNames := make([]string, 0, len(advisoryNames)-len(advisoryMetadata))
	for _, name := range advisoryNames {
		if !found[name] {
			missingNames = append(missingNames, name)
		}
	}
	return missingNames, nil
}

// Changes data stored in system_advisories, in order to match newest evaluation
// Before this methods stores the entries into the system_advisories table, it locks
// advisory_account_data table, so other evaluations don't interfere with this one
func processSystemAdvisories(system *models.SystemPlatform, vmaasData *vmaas.UpdatesV3Response) (
	[]int64, []int64, []int64, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("advisories-processing"))

	deleteIDs, installableNames, applicableNames, err := getAdvisoryChanges(system, vmaasData)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Unable to get system advisory change")
	}
	updatesCnt.WithLabelValues("patched").Add(float64(len(deleteIDs)))
	utils.LogInfo("inventoryID", system.InventoryID, "fixed", len(deleteIDs), "fixed advisories")

	installableIDs, missingInstallableNames, err := getAdvisoriesFromDB(installableNames, system.InventoryID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Unable to ensure new installable system advisories in db")
	}

	missingInstallableIDs, err := storeMissingAdvisories(missingInstallableNames)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to store unknown installable advisories in db")
	}
	installableIDs = append(installableIDs, missingInstallableIDs...)
	updatesCnt.WithLabelValues("installable").Add(float64(len(installableIDs)))
	utils.LogInfo("inventoryID", system.InventoryID, "installable", len(installableIDs), "installable advisories")

	applicableIDs, missingApplicableNames, err := getAdvisoriesFromDB(applicableNames, system.InventoryID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Unable to ensure new applicable system advisories in db")
	}

	missingApplicableIDs, err := storeMissingAdvisories(missingApplicableNames)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to store unknown applicable advisories in db")
	}
	applicableIDs = append(applicableIDs, missingApplicableIDs...)
	updatesCnt.WithLabelValues("applicable").Add(float64(len(applicableIDs)))
	utils.LogInfo("inventoryID", system.InventoryID, "applicable", len(applicableIDs), "applicable advisories")

	return deleteIDs, installableIDs, applicableIDs, nil
}

func storeAdvisoryData(tx *gorm.DB, system *models.SystemPlatform,
	deleteIDs, installableIDs, applicableIDs []int64) (SystemAdvisoryMap, error) {
	if !enableAdvisoryAnalysis {
		utils.LogInfo("advisory analysis disabled, skipping storing")
		return nil, nil
	}

	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("advisories-store"))
	err := updateSystemAdvisoriesTODO(tx, system, deleteIDs, installableIDs, applicableIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update system advisories")
	}
	systemAdvisoriesNew, err := loadSystemAdvisories(tx, system.RhAccountID, system.ID) // reload system advisories after update
	if err != nil {
		return nil, errors.Wrap(err, "Unable to load new system advisories")
	}

	err = updateAdvisoryAccountData(tx, system, deleteIDs, installableIDs, applicableIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update advisory_account_data caches")
	}
	return systemAdvisoriesNew, nil
}

func storeAdvisoryData2(tx *gorm.DB, system *models.SystemPlatform, advisoriesByName ExtendedAdvisoryMap) (
	SystemAdvisoryMap, error) {
	if !enableAdvisoryAnalysis {
		utils.LogInfo("advisory analysis disabled, skipping storing")
		return nil, nil
	}

	// TODO: toto prerobit, aby sa spravit DB update a zaroven aj update v `advisoriesByName`
	// => zbavime sa znovunacitania
	// => co s updateAdvisoryAccountData?

	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("advisories-store"))
	err := updateSystemAdvisoriesTODO(tx, system, deleteIDs, installableIDs, applicableIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update system advisories")
	}
	systemAdvisoriesNew, err := loadSystemAdvisories(tx, system.RhAccountID, system.ID) // reload system advisories after update
	if err != nil {
		return nil, errors.Wrap(err, "Unable to load new system advisories")
	}

	err = updateAdvisoryAccountData(tx, system, deleteIDs, installableIDs, applicableIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update advisory_account_data caches")
	}
	return systemAdvisoriesNew, nil
}

func getAdvisoryChanges(system *models.SystemPlatform, vmaasData *vmaas.UpdatesV3Response) (
	[]int64, []string, []string, error) {
	reported := getReportedAdvisories(vmaasData)
	stored, err := loadSystemAdvisories(database.DB, system.RhAccountID, system.ID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Unable to get system stored advisories")
	}

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
	for storedAdvisoryName, storedAdvisory := range stored {
		if _, found := reported[storedAdvisoryName]; !found {
			// advisory was patched from last evaluation,let's remove it
			deleteIDs = append(deleteIDs, storedAdvisory.AdvisoryID)
		}
	}
	return deleteIDs, installableNames, applicableNames, nil
}

func getAdvisoriesFromDB(advisoryNames []string, inventoryID string) ([]int64, []string, error) {
	advisoryMetadata := make(models.AdvisoryMetadataSlice, 0, len(advisoryNames))
	err := database.DB.Model(&models.AdvisoryMetadata{}).
		Where("name IN (?)", advisoryNames).
		Select("id, name").
		Scan(&advisoryMetadata).Error
	if err != nil {
		return nil, nil, err
	}

	found := make(map[string]bool, len(advisoryNames))
	advisoryIDs := make([]int64, 0, len(advisoryNames))
	for _, am := range advisoryMetadata {
		found[am.Name] = true
		advisoryIDs = append(advisoryIDs, am.ID)
	}
	nUnknown := len(advisoryNames) - len(advisoryIDs)
	if nUnknown > 0 {
		utils.LogInfo("inventoryID", inventoryID, "unknown", nUnknown, "unknown advisories")
		updatesCnt.WithLabelValues("unknown").Add(float64(nUnknown))
	}
	missingAdvisoryNames := make([]string, 0, nUnknown)
	for _, name := range advisoryNames {
		if !found[name] {
			missingAdvisoryNames = append(missingAdvisoryNames, name)
		}
	}
	return advisoryIDs, missingAdvisoryNames, err
}

func storeMissingAdvisories(missingNames []string) ([]int64, error) {
	toStore := make(models.AdvisoryMetadataSlice, 0, len(missingNames))
	for _, name := range missingNames {
		if len(name) > 0 && len(name) < 100 {
			toStore = append(toStore, models.AdvisoryMetadata{
				Name:           name,
				Description:    "Not Available",
				Synopsis:       "Not Available",
				Summary:        "Not Available",
				AdvisoryTypeID: 0,
				RebootRequired: false,
				Synced:         false,
			})
		}
	}
	storedIDs, err := lazySaveAdvisories(toStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save advisory_metadata")
	}
	return storedIDs, nil
}

func lazySaveAdvisories(missing models.AdvisoryMetadataSlice) ([]int64, error) {
	var err error
	ret := make([]int64, 0, len(missing))
	if len(missing) > 0 {
		tx := database.DB.Begin()
		defer tx.Commit()
		err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&missing).Error
		if err != nil {
			return ret, err
		}
		for _, x := range missing {
			ret = append(ret, x.ID)
		}
	}
	return ret, nil
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

func updateSystemAdvisoriesTODO(tx *gorm.DB, system *models.SystemPlatform,
	deleteIDs, installableIDs, applicableIDs []int64) error {
	// this will remove many many old items from our "system_advisories" table
	err := deleteOldSystemAdvisories(tx, system.RhAccountID, system.ID, deleteIDs)
	if err != nil {
		return err
	}

	err = ensureSystemAdvisories(tx, system.RhAccountID, system.ID, installableIDs, applicableIDs)
	if err != nil {
		return err
	}

	return nil
}

func loadSystemAdvisories(tx *gorm.DB, accountID int, systemID int64) (SystemAdvisoryMap, error) {
	var data []models.SystemAdvisories
	err := tx.Preload("Advisory").Find(&data, "system_id = ? AND rh_account_id = ?", systemID, accountID).Error
	if err != nil {
		return nil, err
	}

	systemAdvisories := make(SystemAdvisoryMap, len(data))
	for _, sa := range data {
		systemAdvisories[sa.Advisory.Name] = sa
	}
	return systemAdvisories, nil
}

func updateAdvisoryAccountData(tx *gorm.DB, system *models.SystemPlatform, deleteIDs, installableIDs,
	applicableIDs []int64) error {
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

type ExtendedAdvisory struct {
	Change ChangeType
	models.SystemAdvisories
}

type ExtendedAdvisoryMap map[string]ExtendedAdvisory
