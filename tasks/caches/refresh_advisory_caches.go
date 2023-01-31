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
	err := tasks.WithTx(func(tx *gorm.DB) error {
		return tx.Exec("select refresh_advisory_caches(NULL, NULL)").Error
	})
	if err != nil {
		utils.LogError("err", err.Error(), "Refreshed account advisory caches")
	} else {
		utils.LogInfo("Refreshed account advisory caches")
	}
}
