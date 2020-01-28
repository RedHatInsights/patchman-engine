package listener

import (
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"time"
)

func deleteHandler(event mqueue.PlatformEvent) {
	tStart := time.Now()
	defer messageHandlingDuration.WithLabelValues(EventDelete).Observe(time.Since(tStart).Seconds())

	if event.Type == nil {
		utils.Log().Warn("empty event type received")
		messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedErrorOtherType)
		return
	}

	if *event.Type != "delete" {
		utils.Log("eventType", *event.Type).Warn("non-delete event type received")
		messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedErrorOtherType)
		return
	}

	query := database.Db.Exec("select deleted_inventory_id from delete_system(?)", event.ID)
	err := query.Error
	if err != nil {
		utils.Log("id", event.ID, "err", err.Error()).Error("Could not delete system")
		messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedErrorProcessing)
		return
	}

	utils.Log("id", event.ID, "count", query.RowsAffected).Info("Systems deleted")
	messagesReceivedCnt.WithLabelValues(EventDelete, ReceivedSuccess)
}
