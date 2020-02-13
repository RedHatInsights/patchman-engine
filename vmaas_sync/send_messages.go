package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"context"
	"time"
)

func sendReevaluationMessages() error {
	var inventoryIDs []string
	err := database.Db.Model(&models.SystemPlatform{}).
		Pluck("inventory_id", &inventoryIDs).Error
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, inventoryID := range inventoryIDs {
		sendOneMessage(ctx, inventoryID)
	}
	return nil
}

func sendOneMessage(ctx context.Context, inventoryID string) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageSendDuration)

	utils.Log("inventoryID", inventoryID).Debug("Sending evaluation kafka message")
	event := mqueue.PlatformEvent{
		ID: inventoryID,
	}

	err := (*evalWriter).WriteEvent(ctx, event)
	if err != nil {
		utils.Log("err", err.Error(), "inventoryID", inventoryID).
			Error("inventory id sending to re-evaluate failed")
	}
}
