package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"time"
)

const BatchSize = 4000

func sendReevaluationMessages() error {
	if !enableRecalcMessagesSend {
		utils.Log().Info("Recalc messages sending disabled, skipping...")
		return nil
	}

	var inventoryAIDs []mqueue.InventoryAID
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
	for i := 0; i < len(inventoryAIDs); i += BatchSize {
		end := i + BatchSize
		if end > len(inventoryAIDs) {
			end = len(inventoryAIDs)
		}
		mqueue.SendMessages(base.Context, evalWriter, inventoryAIDs[i:end]...)
	}
	utils.Log("count", len(inventoryAIDs)).Info("systems sent to re-calc")
	return nil
}

func getAllInventoryIDs() ([]mqueue.InventoryAID, error) {
	var inventoryAIDs []mqueue.InventoryAID
	err := database.Db.Model(&models.SystemPlatform{}).
		Select("inventory_id, rh_account_id").
		Order("rh_account_id").
		Scan(&inventoryAIDs).Error
	if err != nil {
		return nil, err
	}
	return inventoryAIDs, nil
}
