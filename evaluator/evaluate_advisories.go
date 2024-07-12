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
	ExtendedAdvisoryMap, error) {
	if !enableAdvisoryAnalysis {
		utils.LogInfo("advisory analysis disabled, skipping lazy saving and loading")
		return nil, nil
	}

	err := lazySaveAdvisories(vmaasData, system.InventoryID)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to store unknown advisories in DB")
	}

	stored, err := loadSystemAdvisories(system.RhAccountID, system.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to load system advisories")
	}

	merged, err := evaluateChanges(vmaasData, stored)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to evaluate advisory changes")
	}

	return merged, nil
}

func pasrseReported(stored SystemAdvisoryMap, reported map[string]int) (ExtendedAdvisoryMap, []string) {
	extendedAdvisories := make(ExtendedAdvisoryMap, len(reported)+len(stored))
	missingNames := make([]string, 0, len(reported))
	for reportedName, reportedStatusID := range reported {
		if storedAdvisory, found := stored[reportedName]; found {
			if reportedStatusID != storedAdvisory.StatusID {
				storedAdvisory.StatusID = reportedStatusID
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
					StatusID: reportedStatusID,
				},
			}
			missingNames = append(missingNames, reportedName)
		}
	}
	return extendedAdvisories, missingNames
}

func loadMissingNamesIDs(missingNames []string, extendedAdvisories *ExtendedAdvisoryMap) error {
	advisoryMetadata, err := getAdvisoryMetadataByNames(missingNames)
	if err != nil {
		return err
	}

	name2AdvisoryID := make(map[string]int64, len(missingNames))
	for _, am := range advisoryMetadata {
		name2AdvisoryID[am.Name] = am.ID
	}

	for _, name := range missingNames {
		if _, found := name2AdvisoryID[name]; !found {
			return errors.New("Failed to evaluate changes because an advisory was not lazy-saved")
		}
		extendedAdvisory := (*extendedAdvisories)[name]
		extendedAdvisory.AdvisoryID = name2AdvisoryID[name]
		(*extendedAdvisories)[name] = extendedAdvisory
	}

	return nil
}

// Set change for advisories that are in `stored` but not in `reported` to Remove.
func parseStored(stored SystemAdvisoryMap, reported map[string]int, extendedAdvisories *ExtendedAdvisoryMap) {
	for storedName, storedAdvisory := range stored {
		if _, found := reported[storedName]; !found {
			(*extendedAdvisories)[storedName] = ExtendedAdvisory{
				Change:           Remove,
				SystemAdvisories: storedAdvisory,
			}
		}
	}
}

// Evaluate changes to all advisories based on `stored` advisories from DB and `reported` advisories from VMaaS.
func evaluateChanges(vmaasData *vmaas.UpdatesV3Response, stored SystemAdvisoryMap) (
	ExtendedAdvisoryMap, error) {
	reported := getReportedAdvisories(vmaasData)
	extendedAdvisories, missingNames := pasrseReported(stored, reported)

	err := loadMissingNamesIDs(missingNames, &extendedAdvisories)
	if err != nil {
		return nil, err
	}

	parseStored(stored, reported, &extendedAdvisories)

	return extendedAdvisories, nil
}

// From names reported by VMaaS, find advisories missing in the DB and lazy-save them.
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
	if nUnknown > 0 {
		utils.LogInfo("inventoryID", inventoryID, "unknown", nUnknown, "unknown advisories")
		updatesCnt.WithLabelValues("unknown").Add(float64(nUnknown))
	} else {
		return nil
	}

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

	if len(toStore) > 0 {
		tx := database.DB.Begin()
		defer tx.Commit()
		err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&toStore).Error
		if err != nil {
			return err
		}
		// FIXME: after creation, toStore will include newly added .ID attributes
	}

	return nil
}

func getAdvisoryMetadataByNames(names []string) (models.AdvisoryMetadataSlice, error) {
	metadata := make(models.AdvisoryMetadataSlice, 0, len(names))
	err := database.DB.Model(&models.AdvisoryMetadata{}).
		Where("name IN (?)", names).
		Select("id, name").
		Scan(&metadata).Error
	return metadata, err
}

// Determine if advisories from DB are properly stored based on advisory metadata existence.
func getMissingAdvisories(advisoryNames []string) ([]string, error) {
	advisoryMetadata, err := getAdvisoryMetadataByNames(advisoryNames)
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

func storeAdvisoryData(tx *gorm.DB, system *models.SystemPlatform, advisoriesByName ExtendedAdvisoryMap) (
	SystemAdvisoryMap, error) {
	if !enableAdvisoryAnalysis {
		utils.LogInfo("advisory analysis disabled, skipping storing")
		return nil, nil
	}

	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("advisories-store"))
	deleteIDs, systemAdvisoriesNew, err := updateSystemAdvisories(tx, system, advisoriesByName)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update system advisories")
	}

	// FIXME: Advisory account data should calculate changes from the before-update data
	err = updateAdvisoryAccountData(tx, system, deleteIDs, systemAdvisoriesNew)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update advisory_account_data caches")
	}
	return systemAdvisoriesNew, nil
}

func calcAdvisoryChanges(system *models.SystemPlatform, deleteIDs []int64,
	systemAdvisories SystemAdvisoryMap) []models.AdvisoryAccountData {
	// If system is stale, we won't change any rows  in advisory_account_data
	if system.Stale {
		return []models.AdvisoryAccountData{}
	}

	aadMap := make(map[int64]models.AdvisoryAccountData, len(systemAdvisories))
	isApplicableOnly := make(map[int64]bool, len(systemAdvisories))
	for _, advisory := range systemAdvisories {
		if advisory.StatusID == INSTALLABLE {
			aadMap[advisory.AdvisoryID] = models.AdvisoryAccountData{
				AdvisoryID:         advisory.AdvisoryID,
				RhAccountID:        system.RhAccountID,
				SystemsInstallable: 1,
				// every installable advisory is also applicable advisory
				SystemsApplicable: 1,
			}
		} else { // APPLICABLE
			isApplicableOnly[advisory.AdvisoryID] = true
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

	for _, id := range deleteIDs {
		aadMap[id] = models.AdvisoryAccountData{
			AdvisoryID:         id,
			RhAccountID:        system.RhAccountID,
			SystemsInstallable: -1,
		}
		if !isApplicableOnly[id] {
			// advisory is no longer applicable
			aad := aadMap[id]
			aad.SystemsApplicable = -1
			aadMap[id] = aad
		}
	}

	deltas := make([]models.AdvisoryAccountData, 0, len(deleteIDs)+len(systemAdvisories))
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

func upsertSystemAdvisories(tx *gorm.DB, advisoryObjs models.SystemAdvisoriesSlice) error {
	tx = database.OnConflictUpdateMulti(tx, []string{"rh_account_id", "system_id", "advisory_id"}, "status_id")
	return database.BulkInsert(tx, advisoryObjs)
}

func processAdvisories(system *models.SystemPlatform, advisoriesByName ExtendedAdvisoryMap) ([]int64,
	models.SystemAdvisoriesSlice, SystemAdvisoryMap) {
	deleteIDs := make([]int64, 0, len(advisoriesByName))
	advisoryObjs := make(models.SystemAdvisoriesSlice, 0, len(advisoriesByName))
	updatedAdvisories := make(SystemAdvisoryMap, len(advisoriesByName))
	for name, advisory := range advisoriesByName {
		switch advisory.Change {
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
			updatedAdvisories[name] = adv
		case Keep:
			updatedAdvisories[name] = advisory.SystemAdvisories
		}
	}
	return deleteIDs, advisoryObjs, updatedAdvisories
}

func updateSystemAdvisories(tx *gorm.DB, system *models.SystemPlatform,
	advisoriesByName ExtendedAdvisoryMap) ([]int64, SystemAdvisoryMap, error) {
	deleteIDs, advisoryObjs, updatedAdvisories := processAdvisories(system, advisoriesByName)

	// this will remove many many old items from our "system_advisories" table
	err := deleteOldSystemAdvisories(tx, system.RhAccountID, system.ID, deleteIDs)
	if err != nil {
		return nil, nil, err
	}

	err = upsertSystemAdvisories(tx, advisoryObjs)
	if err != nil {
		return nil, nil, err
	}

	return deleteIDs, updatedAdvisories, nil
}

func loadSystemAdvisories(accountID int, systemID int64) (SystemAdvisoryMap, error) {
	var data []models.SystemAdvisories
	err := database.DB.Preload("Advisory").Find(&data, "system_id = ? AND rh_account_id = ?", systemID, accountID).Error
	if err != nil {
		return nil, err
	}

	systemAdvisories := make(SystemAdvisoryMap, len(data))
	for _, sa := range data {
		systemAdvisories[sa.Advisory.Name] = sa
	}
	return systemAdvisories, nil
}

func updateAdvisoryAccountData(tx *gorm.DB, system *models.SystemPlatform,
	deleteIDs []int64, systemAdvisories SystemAdvisoryMap) error {
	changes := calcAdvisoryChanges(system, deleteIDs, systemAdvisories)

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

const (
	UnknownFixme int = iota
	Enhancement
	Bugfix
	Security
)
