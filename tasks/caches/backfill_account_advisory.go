package caches

import (
	"app/aggregator"
	"app/base/database"
	"app/base/utils"
	"app/tasks"
	"sync"

	"gorm.io/gorm"
)

func BackfillAccountAdvisory() {
	var wg sync.WaitGroup
	backfillAccountAdvisoryPerAccounts(&wg)
	wg.Wait()
}

func backfillAccountAdvisoryPerAccounts(wg *sync.WaitGroup) {
	var rhAccountIDs []int
	err := tasks.WithReadReplicaTx(func(tx *gorm.DB) error {
		return tx.Table("rh_account").
			Order("hash_partition_id(id, 128), id").
			Pluck("id", &rhAccountIDs).Error
	})
	if err != nil {
		utils.LogError("err", err, "unable to load rh_account IDs for account_advisory backfill")
		return
	}

	utils.LogInfo("accounts", len(rhAccountIDs), "starting account_advisory backfill")

	guard := make(chan struct{}, 4)

	for i, rhAccountID := range rhAccountIDs {
		guard <- struct{}{}
		wg.Add(1)
		go func(i, rhAccountID int) {
			defer func() {
				<-guard
				wg.Done()
			}()

			err := tasks.WithTx(func(tx *gorm.DB) error {
				utils.LogInfo("i", i, "rh_account_id", rhAccountID, "backfilling account_advisory")
				return tx.Exec("SELECT backfill_account_advisory(?)", rhAccountID).Error
			})
			if err != nil {
				utils.LogError("err", err, "rh_account_id", rhAccountID, "failed to backfill account_advisory")
				return
			}
			utils.LogInfo("i", i, "rh_account_id", rhAccountID, "backfilled account_advisory")

			var advisoryIDs []int64
			if err := database.DB.Table("account_advisory").
				Where("rh_account_id = ?", rhAccountID).
				Distinct("advisory_id").
				Pluck("advisory_id", &advisoryIDs).Error; err != nil {
				utils.LogError("err", err, "rh_account_id", rhAccountID, "failed to load advisory IDs for drift check")
				return
			}
			aggregator.CheckAdvisoryDrift(rhAccountID, advisoryIDs)
		}(i, rhAccountID)
	}
}
