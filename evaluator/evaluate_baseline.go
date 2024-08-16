package evaluator

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/vmaas"
	"time"
)

func limitVmaasToBaseline(system *models.SystemPlatform, vmaasData *vmaas.UpdatesV3Response) error {
	baselineConfig := database.GetBaselineConfig(system)
	if baselineConfig == nil {
		return nil // no baseline config, nothing to change
	}

	reportedMap := getReportedAdvisories(vmaasData)
	reportedNames := make([]string, 0, len(reportedMap))
	for name := range reportedMap {
		reportedNames = append(reportedNames, name)
	}

	var filterOutNames []string
	err := database.DB.Model(&models.AdvisoryMetadata{}).Where("name IN (?)", reportedNames).
		Where("public_date >= ?", baselineConfig.ToTime.Truncate(24*time.Hour)).
		Pluck("name", &filterOutNames).Error
	if err != nil {
		return base.WrapFatalDBError(err, "load reported advisories")
	}

	// create map of advisories we need to filter out
	filterOutNamesSet := make(map[string]struct{}, len(filterOutNames))
	for _, i := range filterOutNames {
		filterOutNamesSet[i] = struct{}{}
	}

	updateList := vmaasData.GetUpdateList()
	modifiedUpdateList := make(map[string]*vmaas.UpdatesV3ResponseUpdateList, len(updateList))
	for pkg, updates := range updateList {
		availableUpdates := updates.GetAvailableUpdates()
		for i := range availableUpdates {
			advisoryName := availableUpdates[i].GetErratum()
			if _, ok := filterOutNamesSet[advisoryName]; ok {
				availableUpdates[i].StatusID = APPLICABLE
			} else {
				availableUpdates[i].StatusID = INSTALLABLE
			}
		}
		updates.AvailableUpdates = &availableUpdates
		modifiedUpdateList[pkg] = updates
	}

	if vmaasData != nil && vmaasData.UpdateList != nil {
		vmaasData.UpdateList = &modifiedUpdateList
	}
	return nil
}
