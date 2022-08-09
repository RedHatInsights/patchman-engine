package caches

import (
	"app/base/utils"
	"app/tasks"

	"gorm.io/gorm"
)

func RefreshLatestPackagesView() {
	err := tasks.WithTx(func(tx *gorm.DB) error {
		return tx.Exec("select refresh_latest_packages_view()").Error
	})

	if err != nil {
		utils.Log("err", err.Error()).Error("Refreshing latest_packages_view")
	} else {
		utils.Log().Info("Refreshed latest_packages_view")
	}
}
