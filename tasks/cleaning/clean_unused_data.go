package cleaning

import (
	"app/base/models"
	"app/base/utils"
	"app/tasks"
)

var (
	enableUnusedDataDelete bool
	deleteUnusedDataLimit  int
)

func init() {
	deleteUnusedDataLimit = utils.GetIntEnvOrDefault("DELETE_UNUSED_DATA_LIMIT", 1000)
	enableUnusedDataDelete = utils.GetBoolEnvOrDefault("ENABLE_UNUSED_DATA_DELETE", true)
}

func RunDeleteUnusedData() {
	defer utils.LogPanics(true)
	utils.Log().Info("Deleting unused data")

	deleteUnusedPackages()
	deleteUnusedAdvisories()
}

func deleteUnusedPackages() {
	if !enableUnusedDataDelete {
		return
	}
	tx := tasks.CancelableDB().Begin()
	defer tx.Rollback()

	// remove unused packages not synced from vmaas
	// before changing the query below test its performance on big data otherwise it can lock database
	subq := tx.Select("id").Table("package p").
		Where("synced = ?", false).
		Where("NOT EXISTS (SELECT 1 FROM system_package sp WHERE p.id = sp.package_id)").
		Limit(deleteUnusedDataLimit)

	err := tx.Delete(&models.Package{}, "id IN (?)", subq).Error

	if err != nil {
		utils.Log("err", err.Error()).Error("DeleteUnusedPackages")
		return
	}

	tx.Commit()
	utils.Log().Info("DeleteUnusedPackages tasks performed successfully")
}

func deleteUnusedAdvisories() {
	if !enableUnusedDataDelete {
		return
	}
	tx := tasks.CancelableDB().Begin()
	defer tx.Rollback()

	// remove unused advisories not synced from vmaas
	// before changing the query below test its performance on big data otherwise it can lock database
	// Time: 18988.223 ms (00:18.988) for 50k advisories, 75M system_advisories, 1.6M package and 50k rh_account
	subq := tx.Select("id").Table("advisory_metadata am").
		Where("am.synced = ?", false).
		Where("NOT EXISTS (SELECT 1 FROM system_advisories sa WHERE am.id = sa.advisory_id)").
		Where("NOT EXISTS (SELECT 1 FROM package p WHERE am.id = p.advisory_id)").
		Where("NOT EXISTS (SELECT 1 FROM advisory_account_data aad WHERE am.id = aad.advisory_id)").
		Limit(deleteUnusedDataLimit)

	err := tx.Delete(&models.AdvisoryMetadata{}, "id IN (?)", subq).Error

	if err != nil {
		utils.Log("err", err.Error()).Error("DeleteUnusedAdvisories")
		return
	}

	tx.Commit()
	utils.Log().Info("DeleteUnusedAdvisories tasks performed successfully")
}
