package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/mqueue"
	"app/base/utils"
	"app/tasks"
	"time"
)

func SendReevaluationMessages() error {
	if !tasks.EnableRecalcMessagesSend {
		utils.LogInfo("Recalc messages sending disabled, skipping...")
		return nil
	}

	var inventoryAIDs mqueue.EvalDataSlice
	var err error

	if tasks.EnabledRepoBasedReeval {
		inventoryAIDs, err = getCurrentRepoBasedInventoryIDs()
	} else {
		inventoryAIDs, err = getAllInventoryIDs()
	}
	if err != nil {
		return err
	}

	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageSendDuration)
	err = mqueue.SendMessages(base.Context, evalWriter, &inventoryAIDs)
	if err != nil {
		utils.LogError("err", err.Error(), "sending to re-evaluate failed")
	}
	utils.LogInfo("count", len(inventoryAIDs), "systems sent to re-calc")
	return nil
}

func getAllInventoryIDs() ([]mqueue.EvalData, error) {
	var inventoryAIDs []mqueue.EvalData
	err := tasks.CancelableDB().Table("system_platform sp").
		Select("sp.inventory_id, sp.rh_account_id, ra.org_id").
		Joins("JOIN rh_account ra on ra.id = sp.rh_account_id").
		Order("ra.id").
		Scan(&inventoryAIDs).Error
	if err != nil {
		return nil, err
	}
	return inventoryAIDs, nil
}
