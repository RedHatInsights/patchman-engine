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

// LazySaveAndLoadAdvisories lazy saves missing advisories from reported, loads stored ones from DB,
// and evaluates changes between the two.
func lazySaveAndLoadAdvisories(system *models.SystemPlatform, vmaasData *vmaas.UpdatesV3Response) (
	extendedAdvisoryMap, error) {
	if !enableAdvisoryAnalysis {
		utils.LogInfo("advisory analysis disabled, skipping lazy saving and loading")
		return nil, nil
	}

	err := lazySaveAdvisories(vmaasData, system.InventoryID)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to store unknown advisories in DB")
	}

	stored, err := loadSystemAdvisories(database.DB, system.RhAccountID, system.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to load system advisories")
	}

	merged, err := evaluateChanges(vmaasData, stored)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to evaluate advisory changes")
	}

	return merged, nil
}

func loadMissingNamesIDs(missingNames []string, extendedAdvisories extendedAdvisoryMap) error {
	name2AdvisoryID, err := getAdvisoryMetadataIDs(missingNames)
	if err != nil {
		return err
	}

	for _, name := range missingNames {
		if _, found := name2AdvisoryID[name]; !found {
			return errors.New("Failed to evaluate changes because an advisory was not lazy saved")
		}
		extendedAdvisory := extendedAdvisories[name]
		extendedAdvisory.AdvisoryID = name2AdvisoryID[name]
		extendedAdvisories[name] = extendedAdvisory
	}

	return nil
}

// EvaluateChanges evaluates all types of changes (Keep, Add, Update, Remove)
// between the stored advisories from DB and the reported advisories from VMaaS,
// and tracks them in extendedAdvisoryMap.
func evaluateChanges(vmaasData *vmaas.UpdatesV3Response, stored SystemAdvisoryMap) (
	extendedAdvisoryMap, error) {
	reported := getReportedAdvisories(vmaasData)
	extendedAdvisories := make(extendedAdvisoryMap, len(reported)+len(stored))
	missingNames := make([]string, 0, len(reported))
	for reportedName, reportedStatusID := range reported {
		if storedAdvisory, found := stored[reportedName]; found {
			if reportedStatusID != storedAdvisory.StatusID {
				storedAdvisory.StatusID = reportedStatusID
				extendedAdvisories[reportedName] = extendedAdvisory{
					change:           Update,
					SystemAdvisories: storedAdvisory,
				}
			} else {
				extendedAdvisories[reportedName] = extendedAdvisory{
					change:           Keep,
					SystemAdvisories: storedAdvisory,
				}
			}
		} else {
			extendedAdvisories[reportedName] = extendedAdvisory{
				change: Add,
				SystemAdvisories: models.SystemAdvisories{
					StatusID: reportedStatusID,
				},
			}
			missingNames = append(missingNames, reportedName)
		}
	}

	err := loadMissingNamesIDs(missingNames, extendedAdvisories)
	if err != nil {
		return nil, err
	}

	for storedName, storedAdvisory := range stored {
		if _, found := reported[storedName]; !found {
			extendedAdvisories[storedName] = extendedAdvisory{
				change:           Remove,
				SystemAdvisories: storedAdvisory,
			}
		}
	}

	return extendedAdvisories, nil
}

// LazySaveAdvisories finds advisories reported by VMaaS and missing in the DB and lazy saves them.
func lazySaveAdvisories(vmaasData *vmaas.UpdatesV3Response, inventoryID string) error {
	reportedNames := getReportedAdvisoryNames(vmaasData)
	if len(reportedNames) < 1 {
		return nil
	}

	missingNames, err := getMissingAdvisories(reportedNames)
	if err != nil {
		return errors.Wrap(err, "Unable to get missing system advisories")
	}

	nUnknown := len(missingNames)
	if nUnknown == 0 {
		return nil
	}
	utils.LogInfo("inventoryID", inventoryID, "unknown", nUnknown, "unknown advisories")
	updatesCnt.WithLabelValues("unknown").Add(float64(nUnknown))

	err = storeMissingAdvisories(missingNames)
	if err != nil {
		return errors.Wrap(err, "Failed to save missing advisory_metadata")
	}

	return nil
}

func storeMissingAdvisories(missingNames []string) error {
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

	if len(toStore) == 0 {
		return nil
	}

	tx := database.DB.Begin()
	defer tx.Commit()
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&toStore).Error
}

func getAdvisoryMetadataIDs(names []string) (map[string]int64, error) {
	metadata := make(models.AdvisoryMetadataSlice, 0, len(names))
	err := database.DB.Model(&models.AdvisoryMetadata{}).
		Where("name IN (?)", names).
		Select("id, name").
		Scan(&metadata).Error
	if err != nil {
		return nil, err
	}

	name2AdvisoryID := make(map[string]int64, len(metadata))
	for _, am := range metadata {
		name2AdvisoryID[am.Name] = am.ID
	}
	return name2AdvisoryID, err
}

// GetMissingAdvisories determines if advisories from DB are properly stored based on advisory metadata existence.
func getMissingAdvisories(advisoryNames []string) ([]string, error) {
	name2AdvisoryID, err := getAdvisoryMetadataIDs(advisoryNames)
	if err != nil {
		return nil, err
	}

	missingNames := make([]string, 0, len(advisoryNames)-len(name2AdvisoryID))
	for _, name := range advisoryNames {
		if _, found := name2AdvisoryID[name]; !found {
			missingNames = append(missingNames, name)
		}
	}

	return missingNames, nil
}

func storeAdvisoryData(tx *gorm.DB, system *models.SystemPlatform, advisoriesByName extendedAdvisoryMap) (
	SystemAdvisoryMap, error) {
	if !enableAdvisoryAnalysis {
		utils.LogInfo("advisory analysis disabled, skipping storing")
		return nil, nil
	}

	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("advisories-store"))
	err := updateSystemAdvisories(tx, system, advisoriesByName)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update system advisories")
	}

	// reload all after updates
	systemAdvisoriesNew, err := loadSystemAdvisories(tx, system.RhAccountID, system.ID)
	if err != nil {
		return nil, err
	}

	err = updateAdvisoryAccountData(tx, system, advisoriesByName)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update advisory_account_data caches")
	}
	return systemAdvisoriesNew, nil
}

func calcAdvisoryChanges(system *models.SystemPlatform,
	advisoriesByName extendedAdvisoryMap) []models.AdvisoryAccountData {
	// If system is stale, we won't change any rows in advisory_account_data
	if system.Stale {
		return []models.AdvisoryAccountData{}
	}

	aadMap := make(map[int64]models.AdvisoryAccountData, len(advisoriesByName))
	for _, advisory := range advisoriesByName {
		switch advisory.change {
		case Remove:
			aadMap[advisory.AdvisoryID] = models.AdvisoryAccountData{
				AdvisoryID:         advisory.AdvisoryID,
				RhAccountID:        system.RhAccountID,
				SystemsInstallable: -1,
			}
			if advisory.StatusID != APPLICABLE { // advisory is no longer applicable
				aad := aadMap[advisory.AdvisoryID]
				aad.SystemsApplicable = -1
				aadMap[advisory.AdvisoryID] = aad
			}
		case Keep:
			continue
		case Add:
			fallthrough
		case Update:
			if advisory.StatusID == INSTALLABLE {
				aadMap[advisory.AdvisoryID] = models.AdvisoryAccountData{
					AdvisoryID:         advisory.AdvisoryID,
					RhAccountID:        system.RhAccountID,
					SystemsInstallable: 1,
					// every installable advisory is also applicable advisory
					SystemsApplicable: 1,
				}
			} else { // APPLICABLE
				// add advisories which are only applicable and not installable to `aadMap`
				if _, ok := aadMap[advisory.AdvisoryID]; !ok {
					// FIXME: this check can be removed if advisories don't repeat.
					// Is it possible that there would be 2 advisories with the same AdvisoryID \
					// where one would be one INSTALLABLE and the other APPLICABLE?
					aadMap[advisory.AdvisoryID] = models.AdvisoryAccountData{
						AdvisoryID:        advisory.AdvisoryID,
						RhAccountID:       system.RhAccountID,
						SystemsApplicable: 1,
					}
				}
			}
		}
	}

	// aadMap into aadSlice
	aadSlice := make([]models.AdvisoryAccountData, 0, len(advisoriesByName))
	for _, aad := range aadMap {
		aadSlice = append(aadSlice, aad)
	}
	return aadSlice
}

func deleteOldSystemAdvisories(tx *gorm.DB, accountID int, systemID int64, patched []int64) error {
	err := tx.Where("rh_account_id = ? ", accountID).
		Where("system_id = ?", systemID).
		Where("advisory_id in (?)", patched).
		Delete(&models.SystemAdvisories{}).Error
	return err
}

func upsertSystemAdvisories(tx *gorm.DB, advisoryObjs models.SystemAdvisoriesSlice) error {
	tx = database.OnConflictUpdateMulti(tx, []string{"rh_account_id", "system_id", "advisory_id"}, "status_id")
	return database.BulkInsert(tx, advisoryObjs)
}

func processAdvisories(system *models.SystemPlatform, advisoriesByName extendedAdvisoryMap) ([]int64,
	models.SystemAdvisoriesSlice) {
	deleteIDs := make([]int64, 0, len(advisoriesByName))
	advisoryObjs := make(models.SystemAdvisoriesSlice, 0, len(advisoriesByName))
	for _, advisory := range advisoriesByName {
		switch advisory.change {
		case Remove:
			deleteIDs = append(deleteIDs, advisory.AdvisoryID)
		case Update:
			// StatusID already changed in `evaluateChanges` to reportedStatusID
			fallthrough
		case Add:
			adv := models.SystemAdvisories{
				RhAccountID: system.RhAccountID,
				SystemID:    system.ID,
				AdvisoryID:  advisory.AdvisoryID,
				Advisory:    advisory.Advisory,
				StatusID:    advisory.StatusID,
			}
			advisoryObjs = append(advisoryObjs, adv)
		}
	}
	return deleteIDs, advisoryObjs
}

func updateSystemAdvisories(tx *gorm.DB, system *models.SystemPlatform,
	advisoriesByName extendedAdvisoryMap) error {
	deleteIDs, advisoryObjs := processAdvisories(system, advisoriesByName)

	err := deleteOldSystemAdvisories(tx, system.RhAccountID, system.ID, deleteIDs)
	if err != nil {
		return err
	}

	return upsertSystemAdvisories(tx, advisoryObjs)
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

func updateAdvisoryAccountData(tx *gorm.DB, system *models.SystemPlatform, advisoriesByName extendedAdvisoryMap) error {
	changes := calcAdvisoryChanges(system, advisoriesByName)

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

type extendedAdvisory struct {
	change ChangeType
	models.SystemAdvisories
}

type extendedAdvisoryMap map[string]extendedAdvisory

const (
	undefinedChangeType int = iota
	enhancement
	bugfix
	security
)
