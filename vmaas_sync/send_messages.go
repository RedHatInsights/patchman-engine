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
	var inventoryIDs []string
	var err error

	if enabledRepoBasedReeval {
		inventoryIDs, err = getCurrentRepoBasedInventoryIDs()
	} else {
		inventoryIDs, err = getAllInventoryIDs()
	}
	if err != nil {
		return err
	}

	for i := 0; i < len(inventoryIDs); i += BatchSize {
		end := i + BatchSize
		if end > len(inventoryIDs) {
			end = len(inventoryIDs)
		}
		sendMessages(base.Context, inventoryIDs[i:end]...)
	}
	utils.Log("count", len(inventoryIDs)).Info("systems sent to re-calc")
	return nil
}

func getAllInventoryIDs() ([]string, error) {
	var inventoryIDs []string
	err := database.Db.Model(&models.SystemPlatform{}).
		Pluck("inventory_id", &inventoryIDs).Error
	if err != nil {
		return nil, err
	}
	return inventoryIDs, nil
}

func sendMessages(ctx context.Context, inventoryIDs ...string) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageSendDuration)

	events := make([]mqueue.PlatformEvent, len(inventoryIDs))

	for i, id := range inventoryIDs {
		events[i] = mqueue.PlatformEvent{
			ID: id,
		}
	}

	err := mqueue.WriteEvents(ctx, evalWriter, events...)
	if err != nil {
		utils.Log("err", err.Error()).Error("sending to re-evaluate failed")
	}
}
