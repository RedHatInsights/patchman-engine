package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/database"
	"time"
)

func RunSystemCulling() {
	ticker := time.NewTicker(time.Hour)

	for {
		<-ticker.C

		database.Db.Exec("select delete_culled_systems()")
		database.Db.Exec("select mark_stale_systems()")
	}
}
