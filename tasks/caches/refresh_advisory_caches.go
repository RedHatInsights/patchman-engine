package caches

import (
	"app/base/utils"
	"app/tasks"
	"sync"

	"github.com/pkg/errors"
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
	err := tasks.WithReadReplicaTx(func(tx *gorm.DB) error {
		return tx.Table("rh_account").
			Where("valid_advisory_cache = FALSE").
			Order("hash_partition_id(id, 128), id").
			Pluck("id", &rhAccountIDs).Error
	})
	if skipNAccountsRefresh > 0 {
		utils.LogInfo("n", skipNAccountsRefresh, "Skipping refresh of first N accounts")
		rhAccountIDs = rhAccountIDs[skipNAccountsRefresh:]
	}
	utils.LogInfo("accounts", len(rhAccountIDs), "Starting advisory cache refresh for accounts")
	if err != nil {
		utils.LogError("err", err.Error(), "Unable to load rh_account table ids to refresh caches")
		return
	}

	// use max 4 goroutines for cache refresh
	guard := make(chan struct{}, 4)

	for i, rhAccountID := range rhAccountIDs {
		guard <- struct{}{}
		wg.Add(1)
		go func(i, rhAccountID int) {
			defer func() {
				<-guard
				wg.Done()
			}()

			err = tasks.WithTx(func(tx *gorm.DB) error {
				utils.LogInfo("i", i, "rh_account_id", rhAccountID, "Refreshing account advisory cache")
				return tx.Exec("select refresh_advisory_caches(NULL, ?)", rhAccountID).Error
			})
			if err != nil {
				utils.LogError("err", err.Error(), "rh_account_id", rhAccountID,
					"Refreshed account advisory caches")
				return
			}
			if err := updateAdvisoryCacheValidity(rhAccountID); err != nil {
				utils.LogError("err", err.Error(), "rh_account_id", rhAccountID, "Refresh failed")
				return
			}
			utils.LogInfo("i", i, "rh_account_id", rhAccountID, "Refreshed account advisory cache")
		}(i, rhAccountID)
	}
}

func updateAdvisoryCacheValidity(accID int) error {
	utils.LogDebug("Updating cache validity")
	err := tasks.WithTx(func(tx *gorm.DB) error {
		return tx.Table("rh_account").
			Where("valid_advisory_cache = ?", false).
			Where("id = ?", accID).
			Update("valid_advisory_cache", true).Error
	})
	return errors.Wrap(err, "failed to update valid_advisory_cache")
}
