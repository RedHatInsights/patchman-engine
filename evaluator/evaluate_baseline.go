package evaluator

import (
	"app/base/database"
	"app/base/models"
	"app/base/types/vmaas"
	"time"

	"gorm.io/gorm"
)

func limitVmaasToBaseline(tx *gorm.DB, system *models.SystemPlatform, vmaasData *vmaas.UpdatesV2Response) error {
	baselineConfig := database.GetBaselineConfig(tx, system)
	if baselineConfig == nil {
		return nil // no baseline config, nothing to change
	}

	reportedMap := getReportedAdvisories(vmaasData)
	reportedNames := make([]string, 0, len(reportedMap))
	for name := range reportedMap {
		reportedNames = append(reportedNames, name)
	}

	var filterOutNames []string
	err := tx.Model(&models.AdvisoryMetadata{}).Where("name IN (?)", reportedNames).
		Where("public_date >= ?", baselineConfig.ToTime.Truncate(24*time.Hour)).
		Pluck("name", &filterOutNames).Error
	if err != nil {
		return err
	}

	// create map of advisories we need to filter out
	filterOutNamesSet := make(map[string]struct{}, len(filterOutNames))
	for _, i := range filterOutNames {
		filterOutNamesSet[i] = struct{}{}
	}

	for pkg, updates := range vmaasData.GetUpdateList() {
		availableUpdates := updates.GetAvailableUpdates()
		filteredUpdates := make([]vmaas.UpdatesV2ResponseAvailableUpdates, 0, len(availableUpdates))
		for _, u := range availableUpdates {
			advisoryName := u.GetErratum()
			if _, ok := filterOutNamesSet[advisoryName]; !ok {
				filteredUpdates = append(filteredUpdates, u)
			}
		}
		updates.AvailableUpdates = &filteredUpdates
		(*vmaasData.UpdateList)[pkg] = updates
	}

	return nil
}
