package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/utils"
	"time"
)

func RunSystemCulling() {
	defer utils.LogPanicsAndExit()

	ticker := time.NewTicker(time.Minute * 10)

	for {
		<-ticker.C
		tx := database.Db.BeginTx(base.Context, nil)
		tx.Exec("select delete_culled_systems()")
		tx.Exec("select mark_stale_systems()")

		if err := tx.Commit().Error; err != nil {
			utils.Log("err", err.Error()).Info("Commit of system culling failed")
			tx.RollbackUnlessCommitted()
			return
		}
		utils.Log().Info("System culling tasks performed successfully")
	}
}
