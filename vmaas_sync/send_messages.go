package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"context"
	"time"
)

const BatchSize = 4000

func sendReevaluationMessages() error {
	inventoryIDs, err := getAllInventoryIDs()
	if err != nil {
		return err
	}

	ctx := context.Background()
	for i := 0; i < len(*inventoryIDs); i += BatchSize {
		end := i + BatchSize
		if end > len(*inventoryIDs) {
			end = len(*inventoryIDs)
		}
		sendMessages(ctx, (*inventoryIDs)[i:end]...)
	}
	return nil
}

func getAllInventoryIDs() (*[]string, error) {
	var inventoryIDs []string
	err := database.Db.Model(&models.SystemPlatform{}).
		Pluck("inventory_id", &inventoryIDs).Error
	if err != nil {
		return nil, err
	}
	return &inventoryIDs, nil
}

func sendMessages(ctx context.Context, inventoryIDs ...string) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageSendDuration)

	events := make([]mqueue.PlatformEvent, len(inventoryIDs))

	for i, id := range inventoryIDs {
		utils.Log("inventoryID", id).Debug("Sending evaluation kafka message")
		events[i] = mqueue.PlatformEvent{
			ID: id,
		}
	}

	err := mqueue.WriteEvents(ctx, evalWriter, events...)
	if err != nil {
		utils.Log("err", err.Error()).Error("sending to re-evaluate failed")
	}
}
