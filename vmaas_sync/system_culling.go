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

			// marking systems as "stale"
			nMarked, err := markSystemsStale(tx)
			if err != nil {
				return errors.Wrap(err, "Mark stale")
			}
			utils.Log("nMarked", nMarked).Info("Stale systems marked")
			staleSystemsMarkedCnt.Add(float64(nMarked))

			return nil
		})

		if err != nil {
			utils.Log("err", err.Error()).Error("Systems culling")
		} else {
			utils.Log().Info("Systems culling tasks performed successfully")
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

func markSystemsStale(tx *gorm.DB) (nMarked int, err error) {
	var nMarkedArr []int
	err = tx.Raw("select mark_stale_systems()").
		Pluck("mark_stale_systems", &nMarkedArr).Error
	if len(nMarkedArr) > 0 {
		nMarked = nMarkedArr[0]
	}

	return nMarked, err
}
