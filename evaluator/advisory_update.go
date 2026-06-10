package evaluator

import (
	"app/base"
	"app/base/models"
	"app/base/mqueue"
	"app/base/types"
	"app/base/utils"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var advisoryUpdatePublisher mqueue.Writer

func configureAdvisoryUpdates() {
	if topic := utils.CoreCfg.AdvisoryUpdateTopic; topic != "" {
		advisoryUpdatePublisher = mqueue.NewKafkaWriterFromEnv(topic)
	}
}

func getChangedAdvisoryIDs(advisoriesByName extendedAdvisoryMap) []int64 {
	ids := make([]int64, 0, len(advisoriesByName))
	for _, advisory := range advisoriesByName {
		if advisory.change != Keep {
			ids = append(ids, advisory.AdvisoryID)
		}
	}
	return ids
}

func createAdvisoryUpdateEvent(system *models.SystemPlatformV2, advisoryIDs []int64) mqueue.AdvisoryUpdateEvent {
	var workspaceID uuid.UUID
	if system.Inventory.WorkspaceID != nil {
		workspaceID = *system.Inventory.WorkspaceID
	} else {
		utils.LogWarn("inventoryID", system.GetInventoryID(), "no workspace for system")
	}

	return mqueue.AdvisoryUpdateEvent{
		RhAccountID: system.Inventory.RhAccountID,
		WorkspaceID: workspaceID,
		AdvisoryIDs: advisoryIDs,
		ProducedAt:  types.Rfc3339Timestamp(time.Now()),
	}
}

func publishAdvisoryUpdates(system *models.SystemPlatformV2, advisoriesByName extendedAdvisoryMap) error {
	if advisoryUpdatePublisher == nil {
		return nil
	}

	if len(advisoriesByName) == 0 {
		return nil
	}

	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("advisory-update-publish"))

	// Extract only the changed advisory IDs (delta) for the aggregator
	advisoryIDs := getChangedAdvisoryIDs(advisoriesByName)
	if len(advisoryIDs) == 0 {
		return nil
	}

	event := createAdvisoryUpdateEvent(system, advisoryIDs)
	if err := mqueue.SendMessages(base.Context, advisoryUpdatePublisher, &mqueue.AdvisoryUpdateEvents{event}); err != nil {
		return errors.Wrap(err, "writing advisory update events")
	}

	utils.LogInfo("inventoryID", system.GetInventoryID(), "advisory update event sent successfully")
	return nil
}
