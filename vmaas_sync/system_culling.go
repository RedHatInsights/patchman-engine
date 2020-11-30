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

func RunSystemCulling() {
	defer utils.LogPanics(true)

	ticker := time.NewTicker(time.Minute * 10)

	for {
		<-ticker.C

		err := withTx(func(tx *gorm.DB) error {
			nDeleted, err := deleteCulledSystems(tx, deleteCulledSystemsLimit)
			if err != nil {
				return errors.Wrap(err, "Delete culled")
			}
			utils.Log("nDeleted", nDeleted).Info("Culled systems deleted")

			deletedCulledSystemsCnt.Add(float64(nDeleted))
			if err := tx.Exec("select mark_stale_systems()").Error; err != nil {
				return errors.Wrap(err, "Mark stale")
			}
			return nil
		})

		if err != nil {
			utils.Log("err", err.Error()).Error("System culling")
		} else {
			utils.Log().Info("System culling tasks performed successfully")
		}
	}
}

func deleteCulledSystems(tx *gorm.DB, limitDeleted int) (nDeleted int, err error) {
	var nDeletedArr []int
	err = tx.Raw("select delete_culled_systems(?)", limitDeleted).
		Pluck("delete_culled_systems", &nDeletedArr).Error
	if len(nDeletedArr) > 0 {
		nDeleted = nDeletedArr[0]
	}

	return nDeleted, err
}
