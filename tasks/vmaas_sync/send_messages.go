package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"time"
)

func SendReevaluationMessages() error {
	if !enableRecalcMessagesSend {
		utils.Log().Info("Recalc messages sending disabled, skipping...")
		return nil
	}

	var inventoryAIDs mqueue.InventoryAIDs
	var err error

	if enabledRepoBasedReeval {
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
		utils.Log("err", err.Error()).Error("sending to re-evaluate failed")
	}
	utils.Log("count", len(inventoryAIDs)).Info("systems sent to re-calc")
	return nil
}

func getAllInventoryIDs() ([]mqueue.InventoryAID, error) {
	var inventoryAIDs []mqueue.InventoryAID
	err := database.Db.Table("system_platform sp").
		Select("sp.inventory_id, sp.rh_account_id, ra.org_id").
		Joins("JOIN rh_account ra on ra.id = sp.rh_account_id").
		Order("ra.id").
		Scan(&inventoryAIDs).Error
	if err != nil {
		return nil, err
	}
	return inventoryAIDs, nil
}
