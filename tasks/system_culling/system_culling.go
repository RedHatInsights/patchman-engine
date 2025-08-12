package system_culling

import (
	"app/base/models"
	"app/base/utils"
	"app/tasks"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func runSystemCulling() {
	defer utils.LogPanics(true)

	err := tasks.WithTx(func(tx *gorm.DB) error {
		nDeleted, err := deleteCulledSystems(tx, tasks.DeleteCulledSystemsLimit)
		if err != nil {
			return errors.Wrap(err, "Delete culled")
		}
		utils.LogInfo("nDeleted", nDeleted, "Culled systems deleted")
		deletedCulledSystemsCnt.Add(float64(nDeleted))

		// marking systems as "stale"
		nMarked, err := markSystemsStale(tx, tasks.DeleteCulledSystemsLimit)
		if err != nil {
			return errors.Wrap(err, "Mark stale")
		}
		utils.LogInfo("nMarked", nMarked, "Stale systems marked")
		staleSystemsMarkedCnt.Add(float64(nMarked))

		// pruning deleted_system
		nPruned, err := pruneDeletedSystems(tx, tasks.DeleteCulledSystemsLimit)
		if err != nil {
			return errors.Wrap(err, "Prune deleted_systems")
		}
		utils.LogInfo("nPruned", nPruned, "Deleted_systems items pruned")

		return nil
	})

	if err != nil {
		utils.LogError("err", err.Error(), "System culling")
	} else {
		utils.LogInfo("System culling tasks performed successfully")
	}
}

// https://github.com/go-gorm/gorm/issues/3722
func deleteCulledSystems(tx *gorm.DB, limitDeleted int) (nDeleted int, err error) {
	var nDeletedArr []int
	err = tx.Raw("select delete_culled_systems(?)", limitDeleted).
		Find(&nDeletedArr).Error
	if len(nDeletedArr) > 0 {
		nDeleted = nDeletedArr[0]
	}

	return nDeleted, err
}

func markSystemsStale(tx *gorm.DB, markedLimit int) (nMarked int, err error) {
	var nMarkedArr []int
	err = tx.Raw("select mark_stale_systems(?)", markedLimit).
		Find(&nMarkedArr).Error
	if len(nMarkedArr) > 0 {
		nMarked = nMarkedArr[0]
	}

	return nMarked, err
}

func pruneDeletedSystems(tx *gorm.DB, limitDeleted int) (int64, error) {
	// postgres delete does not support limit
	subQ := tx.Model(&models.DeletedSystem{}).
		Where("when_deleted < ?", time.Now().Add(-tasks.DeletedSystemsThreshold)).
		Limit(limitDeleted).
		Select("inventory_id")
	query := tx.Delete(&models.DeletedSystem{}, "inventory_id in (?)", subQ)
	return query.RowsAffected, query.Error
}
