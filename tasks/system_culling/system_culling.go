package system_culling //nolint:revive,stylecheck

import (
	"app/base/utils"
	"app/tasks"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func runSystemCulling() {
	defer utils.LogPanics(true)

	err := tasks.WithTx(func(tx *gorm.DB) error {
		if enableCulledSystemDelete {
			nDeleted, err := deleteCulledSystems(tx, deleteCulledSystemsLimit)
			if err != nil {
				return errors.Wrap(err, "Delete culled")
			}
			utils.Log("nDeleted", nDeleted).Info("Culled systems deleted")
			deletedCulledSystemsCnt.Add(float64(nDeleted))
		}

		if enableSystemStaling {
			// marking systems as "stale"
			nMarked, err := markSystemsStale(tx, deleteCulledSystemsLimit)
			if err != nil {
				return errors.Wrap(err, "Mark stale")
			}
			utils.Log("nMarked", nMarked).Info("Stale systems marked")
			staleSystemsMarkedCnt.Add(float64(nMarked))
		}

		return nil
	})

	if err != nil {
		utils.Log("err", err.Error()).Error("System culling")
	} else {
		utils.Log().Info("System culling tasks performed successfully")
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
