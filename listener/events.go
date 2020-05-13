package listener

import (
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"encoding/json"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"time"
)

const (
	WarnEmptyEventType = "empty event type received"
	WarnUnknownType    = "unknown event type received"
	WarnNoRowsModified = "no rows modified on delete event"
)

func EventsMessageHandler(m kafka.Message) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageHandlingDuration.WithLabelValues(EventDelete))

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

	switch msgData["type"] {
	case "delete":
		var event mqueue.PlatformEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			utils.Log("inventoryID", msgData["id"], "msg", string(m.Value)).
				Error("Invalid 'delete' message format")
		}
		return HandleDelete(event)
	case "updated":
		var event HostEgressEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			utils.Log("inventoryID", msgData["id"], "msg", string(m.Value)).
				Error("Invalid 'updated' message format")
		}
		return HandleUpdate(event)
	default:
		utils.Log("msg", string(m.Value)).Error(WarnUnknownType)
		return nil
	}
}

func HandleUpdate(event HostEgressEvent) error {
	var system models.SystemPlatform
	err := database.Db.Find(&system, "inventory_id = ?", event.Host.ID).Error
	if err != nil && gorm.IsRecordNotFoundError(err) {
		utils.Log("inventoryID", event.Host.ID).Info("System not found when handling update")
		messagesReceivedCnt.WithLabelValues(EventUpdate, ReceivedWarnNoRows).Inc()
		return nil
	} else if err != nil {
		messagesReceivedCnt.WithLabelValues(EventUpdate, ReceivedErrorOtherType).Inc()
		return errors.Wrap(err, "Loading system to update")
	}
	if event.Host.DisplayName != nil {
		system.DisplayName = *event.Host.DisplayName
	}
	q := database.OnConflictUpdate(database.Db, "inventory_id", "display_name")
	if err := q.Create(&system).Error; err != nil {
		messagesReceivedCnt.WithLabelValues(EventUpdate, ReceivedErrorOtherType).Inc()
		return errors.Wrap(err, "Saving updated system")
	}
	messagesReceivedCnt.WithLabelValues(EventUpdate, RecievedSuccessUpdated).Inc()

	return nil
}

func HandleDelete(event mqueue.PlatformEvent) error {
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

	query := database.Db.Exec("select deleted_inventory_id from delete_system(?)", event.ID)
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
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		utils.Log("inventoryID", event.ID).Warn(WarnNoRowsModified)
		return nil
	}
	return nil
}
