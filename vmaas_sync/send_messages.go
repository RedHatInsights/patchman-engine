package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"context"
	"time"
)

const BatchSize = 4000

func sendReevaluationMessages() error {
	if !enableRecalcMessagesSend {
		utils.Log().Info("Recalc messages sending disabled, skipping...")
		return nil
	}

	var inventoryAIDs []inventoryAID
	var err error

	if enabledRepoBasedReeval {
		inventoryAIDs, err = getCurrentRepoBasedInventoryIDs()
	} else {
		inventoryAIDs, err = getAllInventoryIDs()
	}
	if err != nil {
		return err
	}

	for i := 0; i < len(inventoryAIDs); i += BatchSize {
		end := i + BatchSize
		if end > len(inventoryAIDs) {
			end = len(inventoryAIDs)
		}
		sendMessages(base.Context, inventoryAIDs[i:end]...)
	}
	utils.Log("count", len(inventoryAIDs)).Info("systems sent to re-calc")
	return nil
}

func getAllInventoryIDs() ([]inventoryAID, error) {
	var inventoryAIDs []inventoryAID
	err := database.Db.Model(&models.SystemPlatform{}).
		Select("inventory_id, rh_account_id").
		Scan(&inventoryAIDs).Error
	if err != nil {
		return nil, err
	}
	return inventoryAIDs, nil
}

func sendMessages(ctx context.Context, inventoryAIDs ...inventoryAID) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageSendDuration)

	events := make([]mqueue.PlatformEvent, len(inventoryAIDs))

	now := base.Rfc3339Timestamp(time.Now())

	for i, aid := range inventoryAIDs {
		events[i] = mqueue.PlatformEvent{
			Timestamp: &now,
			ID:        aid.InventoryID,
			AccountID: aid.RhAccountID,
		}
	}

	err := mqueue.WriteEvents(ctx, evalWriter, events...)
	if err != nil {
		utils.Log("err", err.Error()).Error("sending to re-evaluate failed")
	}
}
