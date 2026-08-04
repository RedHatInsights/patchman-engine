package aggregator

import (
	"app/base/database"
	"app/base/utils"
)

type advisoryCounts struct {
	AdvisoryID         int64
	SystemsApplicable  int
	SystemsInstallable int
}

func CheckAdvisoryDrift(rhAccountID int, advisoryIDs []int64) {
	if len(advisoryIDs) == 0 {
		return
	}

	var newCounts []advisoryCounts
	err := database.DB.Table("account_advisory aa").
		Select(`aa.advisory_id,
			SUM(aa.systems_installable) as systems_installable,
			SUM(aa.systems_applicable) as systems_applicable`).
		Where("aa.rh_account_id = ? AND aa.advisory_id IN (?)", rhAccountID, advisoryIDs).
		Group("aa.advisory_id").
		Find(&newCounts).Error
	if err != nil {
		utils.LogError("err", err, "rh_account_id", rhAccountID, "drift check: failed to query account_advisory")
		return
	}

	newCountsMap := make(map[int64]advisoryCounts, len(newCounts))
	for _, c := range newCounts {
		newCountsMap[c.AdvisoryID] = c
	}

	var legacyCounts []advisoryCounts
	err = database.DB.Table("advisory_account_data").
		Select("advisory_id, systems_applicable, systems_installable").
		Where("rh_account_id = ? AND advisory_id IN (?)", rhAccountID, advisoryIDs).
		Find(&legacyCounts).Error
	if err != nil {
		utils.LogError("err", err, "rh_account_id", rhAccountID, "drift check: failed to query advisory_account_data")
		return
	}

	for _, legacy := range legacyCounts {
		newVal, ok := newCountsMap[legacy.AdvisoryID]
		if !ok {
			utils.LogWarn("rh_account_id", rhAccountID, "advisory_id", legacy.AdvisoryID,
				"drift check: advisory present in legacy table but missing from account_advisory")
			continue
		}
		if legacy.SystemsApplicable != newVal.SystemsApplicable || legacy.SystemsInstallable != newVal.SystemsInstallable {
			utils.LogWarn("rh_account_id", rhAccountID, "advisory_id", legacy.AdvisoryID,
				"legacy_applicable", legacy.SystemsApplicable, "new_applicable", newVal.SystemsApplicable,
				"legacy_installable", legacy.SystemsInstallable, "new_installable", newVal.SystemsInstallable,
				"drift check: count mismatch between legacy and new table")
		}
		delete(newCountsMap, legacy.AdvisoryID)
	}

	for _, newVal := range newCountsMap {
		utils.LogWarn("rh_account_id", rhAccountID, "advisory_id", newVal.AdvisoryID,
			"drift check: advisory present in account_advisory but missing from legacy table")
	}
}
