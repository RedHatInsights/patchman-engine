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

// systems are deleted in independent transactions to avoid locking multiple rows for long time
func deleteCulledSystems(tx *gorm.DB, limitDeleted int) (nDeleted int64, err error) {
	var inventoryIDs []string
	err = tx.Model(&models.SystemInventory{}).
		Where("culled_timestamp < ?", time.Now()).
		Order("id").
		Limit(limitDeleted).
		Pluck("inventory_id", &inventoryIDs).Error
	if err != nil {
		return 0, err
	}

	for _, id := range inventoryIDs {
		var rowsAffected int64
		delErr := tasks.CancelableDB().Transaction(func(tx2 *gorm.DB) error {
			res := tx2.Exec("select delete_system(?::uuid)", id)
			if res.Error != nil {
				return res.Error
			}
			rowsAffected = res.RowsAffected
			return nil
		})
		if delErr != nil {
			utils.LogWarn("inventoryID", id, "err", delErr.Error(), "Delete culled system")
			continue
		}
		nDeleted += rowsAffected
	}

	return nDeleted, nil
}

// each update runs in its own transaction to avoid holding locks across many rows
func markSystemsStale(tx *gorm.DB, markedLimit int) (nMarked int64, err error) {
	var candidates []struct {
		RhAccountID int   `gorm:"column:rh_account_id"`
		ID          int64 `gorm:"column:id"`
		Expired     bool  `gorm:"column:expired"`
	}
	now := time.Now()
	err = tx.Model(&models.SystemInventory{}).
		Select("rh_account_id, id, (stale_warning_timestamp < ?) as expired", now).
		Where("stale != (stale_warning_timestamp < ?)", now).
		Order("rh_account_id").Order("id").
		Limit(markedLimit).
		Find(&candidates).Error
	if err != nil {
		return 0, err
	}

	for _, c := range candidates {
		var rowsAffected int64
		markErr := tasks.CancelableDB().Transaction(func(tx2 *gorm.DB) error {
			res := tx2.Model(&models.SystemInventory{}).
				Where("rh_account_id = ? AND id = ?", c.RhAccountID, c.ID).
				Update("stale", c.Expired)
			if res.Error != nil {
				return res.Error
			}
			rowsAffected = res.RowsAffected
			return nil
		})
		if markErr != nil {
			utils.LogWarn("rhAccountID", c.RhAccountID, "systemID", c.ID, "err", markErr.Error(), "Mark stale system")
			continue
		}
		nMarked += rowsAffected
	}

	return nMarked, nil
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
