package caches

import (
	"app/tasks"

	"gorm.io/gorm"
)

func RefreshPackagesCaches(accID *int) error {
	err := tasks.WithTx(func(tx *gorm.DB) error {
		if accID != nil {
			return tx.Exec("select refresh_packages_caches(?)", *accID).Error
		}
		return tx.Exec("select refresh_packages_caches(NULL)").Error
	})
	return err
}
