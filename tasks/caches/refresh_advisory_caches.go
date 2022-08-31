package caches

import (
	"app/base/utils"
	"app/tasks"

	"gorm.io/gorm"
)

func RefreshAdvisoryCaches() {
	if !enableRefreshAdvisoryCaches {
		return
	}
	refreshAdvisoryCachesPerAccounts()
}

func refreshAdvisoryCachesPerAccounts() {
	utils.Log().Info("Refreshing advisory cache")
	err := tasks.WithTx(func(tx *gorm.DB) error {
		return tx.Exec("select refresh_advisory_caches(NULL, NULL)").Error
	})
	if err != nil {
		utils.Log("err", err.Error()).Error("Refreshed account advisory caches")
	} else {
		utils.Log().Info("Refreshed account advisory caches")
	}
}
