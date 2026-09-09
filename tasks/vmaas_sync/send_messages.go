package vmaas_sync

import (
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"app/tasks"
	"context"
	"time"

	"gorm.io/gorm"
)

func SendReevaluationMessages(ctx context.Context) error {
	if !tasks.EnableRecalcMessagesSend {
		utils.LogInfo("Recalc messages sending disabled, skipping...")
		return nil
	}

	var inventoryAIDs mqueue.EvalDataSlice
	var err error
	tx := database.DB(ctx)
	if tasks.EnabledRepoBasedReeval {
		inventoryAIDs, err = getCurrentRepoBasedInventoryIDs(tx)
	} else {
		inventoryAIDs, err = getAllInventoryIDs(tx)
	}
	if err != nil {
		return err
	}

	defer utils.ObserveSecondsSince(time.Now(), messageSendDuration)
	err = mqueue.SendMessages(ctx, evalWriter, &inventoryAIDs)
	if err != nil {
		utils.LogError("err", err, "sending to re-evaluate failed")
	}
	utils.LogInfo("count", len(inventoryAIDs), "systems sent to re-calc")
	return nil
}

func getAllInventoryIDs(tx *gorm.DB) ([]mqueue.EvalData, error) {
	var inventoryAIDs []mqueue.EvalData
	err := tx.Table("system_inventory si").
		Select("si.inventory_id, si.rh_account_id, ra.org_id").
		Joins("JOIN rh_account ra on ra.id = si.rh_account_id").
		Order("ra.id").
		Scan(&inventoryAIDs).Error
	if err != nil {
		return nil, err
	}
	return inventoryAIDs, nil
}
