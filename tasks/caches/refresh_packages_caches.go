package caches

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/tasks"

	"github.com/pkg/errors"

	"gorm.io/gorm"
)

func RefreshPackagesCaches(accID *int) error {
	var err error
	var accs []int
	pkgSysCounts := make([]models.PackageAccountData, 0)

	if accID == nil {
		accs, err = accountsWithoutCache()
		if err != nil {
			return err
		}
	} else {
		accs = append(accs, *accID)
	}

	utils.LogInfo("count", len(accs), "Refreshing accounts")
	for i, acc := range accs {
		acc := acc
		utils.LogInfo("account", acc, "#", i, "Refreshing account")
		if err = getCounts(&pkgSysCounts, &acc); err != nil {
			utils.LogError("err", err.Error(), "Refresh failed")
			continue
		}

		if err = updatePackageAccountData(pkgSysCounts); err != nil {
			utils.LogError("err", err.Error(), "Refresh failed")
			continue
		}

		if err = deleteOldCache(pkgSysCounts, &acc); err != nil {
			utils.LogError("err", err.Error(), "Refresh failed")
			continue
		}

		if err = updateCacheValidity(&acc); err != nil {
			utils.LogError("err", err.Error(), "Refresh failed")
			continue
		}
	}

	return err
}

func accountsWithoutCache() ([]int, error) {
	accs := []int{}
	// order account ids by partition number in system_package (128 partitions)
	// so that we read data sequentially and not jump from one partition to another and back
	err := tasks.WithTx(func(tx *gorm.DB) error {
		return tx.Table("rh_account").
			Select("id").
			Where("valid_package_cache = FALSE").
			Order("hash_partition_id(id, 128), id").
			Find(&accs).Error
	})
	return accs, errors.Wrap(err, "failed to get accounts without cache")
}

func getCounts(pkgSysCounts *[]models.PackageAccountData, accID *int) error {
	err := tasks.WithTx(func(tx *gorm.DB) error {
		q := tx.Table("system_platform sp").
			Select(`
				sp.rh_account_id rh_account_id,
				spkg.name_id package_name_id,
				count(*) as systems_installed,
				count(*) filter (where update_status(spkg.update_data) = 'Installable') as systems_installable,
				count(*) filter (where update_status(spkg.update_data) != 'None') as systems_applicable
			`).
			Joins("JOIN system_package spkg ON sp.id = spkg.system_id AND sp.rh_account_id = spkg.rh_account_id").
			Joins("JOIN rh_account acc ON sp.rh_account_id = acc.id").
			Joins("JOIN inventory.hosts ih ON sp.inventory_id = ih.id").
			Where("sp.packages_installed > 0 AND sp.stale = FALSE").
			Group("sp.rh_account_id, spkg.name_id").
			Order("sp.rh_account_id, spkg.name_id")
		if accID != nil {
			q.Where("sp.rh_account_id = ?", *accID)
		} else {
			q.Where("acc.valid_package_cache = FALSE")
		}
		return q.Find(pkgSysCounts).Error
	})
	return errors.Wrap(err, "failed to get counts")
}

func updatePackageAccountData(pkgSysCounts []models.PackageAccountData) error {
	err := tasks.WithTx(func(tx *gorm.DB) error {
		return database.UnnestInsert(tx,
			"INSERT INTO package_account_data"+
				" (rh_account_id, package_name_id, systems_installed, systems_installable, systems_applicable)"+
				" (SELECT * FROM unnest($1::int[], $2::bigint[], $3::int[], $4::int[], $5::int[]))"+
				" ON CONFLICT (rh_account_id, package_name_id) DO UPDATE SET"+
				" systems_installable = EXCLUDED.systems_installable,"+
				" systems_applicable = EXCLUDED.systems_applicable,"+
				" systems_installed = EXCLUDED.systems_installed", pkgSysCounts)
	})
	return errors.Wrap(err, "failed to insert to package_account_data")
}

func deleteOldCache(pkgSysCounts []models.PackageAccountData, accID *int) error {
	seen := make(map[int]bool, len(pkgSysCounts))
	accs := make([]int, 0, len(pkgSysCounts))
	pkgAcc := make([][]interface{}, 0, len(pkgSysCounts))
	for _, c := range pkgSysCounts {
		pkgAcc = append(pkgAcc, []interface{}{c.PkgNameID, c.AccID})
		if !seen[c.AccID] {
			seen[c.AccID] = true
			accs = append(accs, c.AccID)
		}
	}
	err := tasks.WithTx(func(tx *gorm.DB) error {
		tx = tx.Table("package_account_data").
			Where("(package_name_id, rh_account_id) NOT IN ?", pkgAcc)
		if accID != nil {
			tx = tx.Where("rh_account_id = ?", *accID)
		} else {
			tx = tx.Where("rh_account_id IN ?", accs)
		}
		tx = tx.Delete(&models.PackageAccountData{})
		return tx.Error
	})
	return errors.Wrap(err, "failed to insert to package_account_data")
}

func updateCacheValidity(accID *int) error {
	err := tasks.WithTx(func(tx *gorm.DB) error {
		tx = tx.Table("rh_account acc").
			Where("valid_package_cache = ?", false)
		if accID != nil {
			tx = tx.Where("acc.id = ?", *accID)
		}
		tx = tx.Update("valid_package_cache", true)
		return tx.Error
	})
	return errors.Wrap(err, "failed to update valid_package_cache")
}
