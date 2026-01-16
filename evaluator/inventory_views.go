package evaluator

import (
	"app/base"
	"app/base/inventory_views"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

var patchAppHeader = kafka.Header{Key: "application", Value: []byte("patch")}

var inventoryViewsPublisher mqueue.Writer

func configureInventoryViews() {
	if topic := utils.CoreCfg.InventoryViewsTopic; topic != "" {
		inventoryViewsPublisher = mqueue.NewKafkaWriterFromEnv(topic)
	}
}

func publishInventoryViewsEvent(tx *gorm.DB, systems []models.SystemPlatform, origin *mqueue.PlatformEvent) error {
	if inventoryViewsPublisher == nil {
		return nil
	}

	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("inventory-views-publish"))

	orgID := origin.GetOrgID()
	var requestID string
	if len(origin.RequestIDs) > 0 {
		requestID = origin.RequestIDs[0]
	} else {
		requestID = fmt.Sprintf("patch-%d", time.Now().UnixNano())
	}

	value, err := inventory_views.MakeInventoryViewsEvent(tx, orgID, systems)
	if err != nil {
		return errors.Wrap(err, "creating inventory views event failed")
	}

	headers := []kafka.Header{patchAppHeader, {Key: "request_id", Value: []byte(requestID)}}

	msg, err := mqueue.MessageFromJSON(orgID, value, headers)
	if err != nil {
		return errors.Wrap(err, "creating inventory views message failed")
	}

	err = inventoryViewsPublisher.WriteMessages(base.Context, msg)
	if err != nil {
		return errors.Wrap(err, "writing message to inventory views publisher failed")
	}

	// log the event
	systemIDs := make([]int64, len(systems))
	for i, s := range systems {
		systemIDs[i] = s.ID
	}

	utils.LogInfo("rh_account_ID", systems[0].RhAccountID, "systemIDs", systemIDs,
		"inventory views event sent successfully")

	return nil
}
