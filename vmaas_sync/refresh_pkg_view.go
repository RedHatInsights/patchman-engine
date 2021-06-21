package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/utils"
	"gorm.io/gorm"
	"time"
)

func refreshLatestPackagesCount() {
	if !enableRefreshPackagesCache {
		return
	}

	defer utils.LogPanics(true)

	nWaitSeconds := utils.GetIntEnvOrDefault("REFRESH_PKG_COUNTS_VIEW_SECONDS", 60 * 10) // 10 min

	for {
		refreshLatestPackagesView(nWaitSeconds)
	}
}

func refreshLatestPackagesView(nWaitSeconds int) {
	err := withTx(func(tx *gorm.DB) error {
		return tx.Exec("select refresh_latest_packages_view()").Error
	})

	if err != nil {
		utils.Log("err", err.Error()).Error("Refreshing latest_packages_view")
	} else {
		utils.Log().Info("Refreshed latest_packages_view")
	}
	time.Sleep(time.Second * time.Duration(nWaitSeconds))
}
