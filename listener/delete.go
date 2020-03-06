package listener

import (
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"time"
)

var DeleteMessageHandler = mqueue.MakeMessageHandler(deleteHandler)

const (
	WarnEmptyEventType = "empty event type received"
	WarnNoDeleteType   = "non-delete event type received"
	WarnNoRowsModified = "no rows modified on delete event"
)

func deleteHandler(event mqueue.PlatformEvent) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messageHandlingDuration.WithLabelValues(EventDelete))

	if event.Type == nil {
		utils.Log("inventoryID", event.ID).Warn(WarnEmptyEventType)
		messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedErrorOtherType).Inc()
		return
	}

	if *event.Type != "delete" {
		utils.Log("inventoryID", event.ID, "eventType", *event.Type).Warn(WarnNoDeleteType)
		messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedErrorOtherType).Inc()
		return
	}

	query := database.Db.Exec("select deleted_inventory_id from delete_system(?)", event.ID)
	err := query.Error
	if err != nil {
		utils.Log("inventoryID", event.ID, "err", err.Error()).Error("Could not delete system")
		messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedErrorProcessing).Inc()
		return
	}

	if query.RowsAffected == 0 {
		utils.Log("inventoryID", event.ID).Warn(WarnNoRowsModified)
		messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedErrorNoRows).Inc()
		return
	}

	utils.Log("inventoryID", event.ID, "count", query.RowsAffected).Info("Systems deleted")
	messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedSuccess).Inc()
}
