package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/utils"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"time"
)

// Need to run code within a function, because defer can't be used in loops
func withTx(do func(db *gorm.DB) error) error {
	tx := database.Db.BeginTx(base.Context, nil)
	defer tx.RollbackUnlessCommitted()
	if err := do(tx); err != nil {
		return err
	}
	return errors.Wrap(tx.Commit().Error, "Commit")
}

func performCulling() error {
	return withTx(func(tx *gorm.DB) error {
		if err := tx.Exec("select delete_culled_systems()").Error; err != nil {
			return errors.Wrap(err, "Delete culled")
		}
		if err := tx.Exec("select mark_stale_systems()").Error; err != nil {
			return errors.Wrap(err, "Mark stale")
		}
		return nil
	})
}

func RunSystemCulling() {
	defer utils.LogPanics(true)

	ticker := time.NewTicker(time.Minute * 10)

	for {
		<-ticker.C

		if err := performCulling(); err != nil {
			utils.Log("err", err.Error()).Error("System culling")
		} else {
			utils.Log().Info("System culling tasks performed successfully")
		}
	}
}
