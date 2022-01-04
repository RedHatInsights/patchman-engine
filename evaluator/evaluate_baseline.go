package evaluator

import (
	"app/base/database"
	"app/base/models"
	"time"

	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func limitVmaasToBaseline(tx *gorm.DB, system *models.SystemPlatform, vmaasData *vmaas.UpdatesV2Response) error {
	baseline, err := database.GetBaselineConfig(tx, system)
	if err != nil {
		return errors.Wrap(err, "Failed to read system's baseline")
	}
	if baseline == nil {
		// no baseline, nothing to change
		return nil
	}

	reportedMap := getReportedAdvisories(vmaasData)
	reportedNames := make([]string, 0, len(reportedMap))
	for name := range reportedMap {
		reportedNames = append(reportedNames, name)
	}

	var filterOutNames []string
	err = tx.Model(&models.AdvisoryMetadata{}).Where("name IN (?)", reportedNames).
		Where("public_date >= ?", baseline.ToTime.Truncate(24*time.Hour)).
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
		updates.SetAvailableUpdates(filteredUpdates)
		(*vmaasData.UpdateList)[pkg] = updates
	}

	return nil
}
