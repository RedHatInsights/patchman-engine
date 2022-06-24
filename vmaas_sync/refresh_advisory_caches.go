package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/database"
	"app/base/utils"
	"time"

	"gorm.io/gorm"
)

func refreshAdvisoryCaches() {
	if !enableRefreshAdvisoryCaches {
		return
	}

	nSecondsWait := utils.GetIntEnvOrDefault("ADVISORY_CACHES_WAIT_SECONDS", 60*15) // set 15 min
	for {
		refreshAdvisoryCachesPerAccounts(nSecondsWait)
	}
}

func refreshAdvisoryCachesPerAccounts(nSecondsWait int) {
	var rhAccountIDs []int
	err := database.Db.Table("rh_account").Pluck("id", &rhAccountIDs).Error
	if err != nil {
		utils.Log("err", err.Error()).Error("Unable to load rh_account table ids to refresh caches")
		return
	}

	for _, rhAccountID := range rhAccountIDs {
		time.Sleep(time.Second * time.Duration(nSecondsWait))
		err = withTx(func(tx *gorm.DB) error {
			return tx.Exec("select refresh_advisory_caches(NULL, ?)", rhAccountID).Error
		})
		if err != nil {
			utils.Log("err", err.Error(), "rh_account_id", rhAccountID).
				Error("Refreshed account advisory caches")
		} else {
			utils.Log("rh_account_id", rhAccountID).Info("Refreshed account advisory caches")
		}
	}
}
