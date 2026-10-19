package evaluator

import (
	"app/base/inventory_views"
	"app/base/models"
	"app/base/mqueue"
	"app/base/telemetry"
	"app/base/utils"
	"context"
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

func publishInventoryViewsEvent(ctx context.Context, tx *gorm.DB,
	systems []models.SystemPlatformV2, origin *mqueue.PlatformEvent) error {
	if inventoryViewsPublisher == nil {
		return nil
	}

	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("inventory-views-publish"))

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

	producerCtx, span := telemetry.ProducerContext(ctx, utils.CoreCfg.InventoryViewsTopic, nil)
	err = inventoryViewsPublisher.WriteMessages(producerCtx, msg)
	telemetry.End(span, err)
	if err != nil {
		return errors.Wrap(err, "writing message to inventory views publisher failed")
	}

	// log the event
	systemIDs := make([]int64, len(systems))
	for i, s := range systems {
		systemIDs[i] = s.InternalSystemID()
	}

	utils.LogInfo("rh_account_ID", systems[0].Inventory.RhAccountID, "systemIDs", systemIDs,
		"inventory views event sent successfully")

	return nil
}
