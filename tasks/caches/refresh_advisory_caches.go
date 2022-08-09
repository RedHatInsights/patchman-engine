package caches

import (
	"app/base/database"
	"app/base/utils"
	"app/tasks"
	"sync"

	"gorm.io/gorm"
)

func RefreshAdvisoryCaches() {
	if !enableRefreshAdvisoryCaches {
		return
	}

	var wg sync.WaitGroup
	refreshAdvisoryCachesPerAccounts(&wg)
	wg.Wait()
}

func refreshAdvisoryCachesPerAccounts(wg *sync.WaitGroup) {
	var rhAccountIDs []int
	err := database.Db.Table("rh_account").Pluck("id", &rhAccountIDs).Error
	if err != nil {
		utils.Log("err", err.Error()).Error("Unable to load rh_account table ids to refresh caches")
		return
	}

	for _, rhAccountID := range rhAccountIDs {
		wg.Add(1)
		go func(rhAccountID int) {
			defer wg.Done()
			err = tasks.WithTx(func(tx *gorm.DB) error {
				return tx.Exec("select refresh_advisory_caches(NULL, ?)", rhAccountID).Error
			})
			if err != nil {
				utils.Log("err", err.Error(), "rh_account_id", rhAccountID).
					Error("Refreshed account advisory caches")
			} else {
				utils.Log("rh_account_id", rhAccountID).Info("Refreshed account advisory caches")
			}
		}(rhAccountID)
	}
}
