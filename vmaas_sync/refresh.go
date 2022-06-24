package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/utils"
	"time"

	"gorm.io/gorm"
)

func refreshLatestPackagesView() {
	err := withTx(func(tx *gorm.DB) error {
		return tx.Exec("select refresh_latest_packages_view()").Error
	})

	if err != nil {
		utils.Log("err", err.Error()).Error("Refreshing latest_packages_view")
	} else {
		utils.Log().Info("Refreshed latest_packages_view")
	}
}

func refreshLatestPackagesCount() {
	if !enableRefreshPackagesCache {
		return
	}

	defer utils.LogPanics(true)

	ticker := time.NewTicker(time.Minute * 10)

	for {
		<-ticker.C
		refreshLatestPackagesView()
	}
}
