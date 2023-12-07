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
	utils.LogInfo("Deleting unused data")

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
		Where(`NOT EXISTS (SELECT 1
							 FROM (SELECT package_name_id,
										  jsonb_path_query(update_data, '$.*') as update_data
									 FROM package_system_data) as psd
							WHERE p.name_id = psd.package_name_id
							  AND p.evra in (psd.update_data->>'installed', psd.update_data->>'installable',
											 psd.update_data->>'applicable')
						  )`).
		Limit(deleteUnusedDataLimit)

	err := tx.Delete(&models.Package{}, "id IN (?)", subq).Error

	if err != nil {
		utils.LogError("err", err.Error(), "DeleteUnusedPackages")
		return
	}

	tx.Commit()
	utils.LogInfo("DeleteUnusedPackages tasks performed successfully")
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
		utils.LogError("err", err.Error(), "DeleteUnusedAdvisories")
		return
	}

	tx.Commit()
	utils.LogInfo("DeleteUnusedAdvisories tasks performed successfully")
}
