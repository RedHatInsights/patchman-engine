package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/database"
	"time"
)

func RunSystemCulling() {
	ticker := time.NewTicker(time.Hour)

	for {
		<-ticker.C

		database.Db.Exec("select culled from delete_culled_systems()")
	}
}
