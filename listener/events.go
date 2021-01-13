package listener

import (
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
	"time"
)

const (
	WarnEmptyEventType = "empty event type received"
	WarnUnknownType    = "unknown event type received"
	WarnNoRowsModified = "no rows modified on delete event"
)

func EventsMessageHandler(m kafka.Message) error {
	var msgData map[string]interface{}
	if err := json.Unmarshal(m.Value, &msgData); err != nil {
		utils.Log("msg", string(m.Value)).Error("message is not a valid JSON")
		// Skip invalid messages
		return nil
	}
	if msgData["type"] == nil {
		utils.Log("inventoryID", msgData["id"]).Warn(WarnEmptyEventType)
		messagesReceivedCnt.WithLabelValues("", ReceivedErrorOtherType).Inc()
		return nil
	}

	if enableBypass {
		utils.Log("inventoryID", msgData["id"]).Info("Processing bypassed")
		messagesReceivedCnt.WithLabelValues(msgData["type"].(string), ReceivedBypassed).Inc()
		return nil
	}

	switch msgData["type"] {
	case "delete":
		var event mqueue.PlatformEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			utils.Log("inventoryID", msgData["id"], "msg", string(m.Value)).
				Error("Invalid 'delete' message format")
		}
		return HandleDelete(event)
	case "updated":
		fallthrough
	case "created":
		var event HostEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			utils.Log("inventoryID", msgData["id"], "msg", string(m.Value)).
				Error("Invalid 'updated' message format")
		}
		return HandleUpload(event)
	default:
		utils.Log("msg", string(m.Value)).Warn(WarnUnknownType)
		return nil
	}
}

func HandleDelete(event mqueue.PlatformEvent) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageHandlingDuration.WithLabelValues(EventDelete))
	// TODO: Do we need locking here ?
	err := database.OnConflictUpdate(database.Db, "inventory_id", "when_deleted").
		Save(models.DeletedSystem{
			InventoryID: event.ID,
			WhenDeleted: time.Now(),
		}).Error

	if err != nil {
		utils.Log("inventoryID", event.ID, "err", err.Error()).Error("Could not delete system")
		messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedErrorProcessing).Inc()
		return errors.Wrap(err, "Could not delete system")
	}

	query := database.Db.Exec("select deleted_inventory_id from delete_system(?::uuid)", event.ID)
	err = query.Error
	if err != nil {
		utils.Log("inventoryID", event.ID, "err", err.Error()).Error("Could not delete system")
		messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedErrorProcessing).Inc()
		return errors.Wrap(err, "Could not opt_out system")
	}

	if query.RowsAffected == 0 {
		utils.Log("inventoryID", event.ID).Warn(WarnNoRowsModified)
		messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedWarnNoRows).Inc()
		return nil
	}

	utils.Log("inventoryID", event.ID, "count", query.RowsAffected).Info("Systems deleted")
	messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedSuccess).Inc()

	err = database.Db.
		Delete(&models.DeletedSystem{}, "when_deleted < ?", time.Now().Add(-DeletionThreshold)).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		utils.Log("inventoryID", event.ID).Warn(WarnNoRowsModified)
		return nil
	}
	return nil
}
