package caches

import (
	"app/base/database"
	"app/base/models"
	"app/tasks"

	"github.com/pkg/errors"

	"gorm.io/gorm"
)

func RefreshPackagesCaches(accID *int) error {
	var err error
	pkgSysCounts := make([]models.PackageAccountData, 0)

	if err = getCounts(&pkgSysCounts, accID); err != nil {
		return err
	}

	if err = updatePackageAccountData(pkgSysCounts); err != nil {
		return err
	}

	if err = deleteOldCache(pkgSysCounts, accID); err != nil {
		return err
	}

	err = updateCacheValidity(accID)
	return err
}

func getCounts(pkgSysCounts *[]models.PackageAccountData, accID *int) error {
	err := tasks.WithTx(func(tx *gorm.DB) error {
		q := tx.Table("system_platform sp").
			Select(`
				sp.rh_account_id rh_account_id,
				spkg.name_id package_name_id,
				count(spkg.system_id) as systems_installed,
				count(spkg.system_id) filter (where spkg.latest_evra IS NOT NULL) as systems_updatable
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
		tx = database.OnConflictUpdateMulti(
			tx, []string{"package_name_id", "rh_account_id"}, "systems_updatable", "systems_installed",
		)
		return database.BulkInsert(tx, pkgSysCounts)
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
