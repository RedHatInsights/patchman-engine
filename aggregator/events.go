package aggregator

import (
	"app/base"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/lib/pq"
)

var (
	batchSize    int
	flushTimeout time.Duration
)

func advisoryUpdateHandler(ctx context.Context, msgs []mqueue.KafkaMessage) error {
	if enableBypass {
		utils.LogWarn("Advisory update processing bypassed")
		return nil
	}
	events := make([]mqueue.AdvisoryUpdateEvent, 0, len(msgs))
	for _, m := range msgs {
		var event mqueue.AdvisoryUpdateEvent
		if err := sonic.Unmarshal(m.Value, &event); err != nil {
			utils.LogError("err", err, "could not deserialize advisory update event")
			continue
		}
		events = append(events, event)
	}
	if len(events) == 0 {
		return nil
	}
	return processAdvisoryBatch(ctx, groupAdvisoryUpdates(events))
}

func groupAdvisoryUpdates(events []mqueue.AdvisoryUpdateEvent) map[int][]int64 {
	sets := make(map[int]map[int64]struct{})
	for _, e := range events {
		if _, ok := sets[e.RhAccountID]; !ok {
			sets[e.RhAccountID] = make(map[int64]struct{})
		}
		for _, id := range e.AdvisoryIDs {
			sets[e.RhAccountID][id] = struct{}{}
		}
	}

	grouped := make(map[int][]int64, len(sets))
	for accID, idSet := range sets {
		ids := make([]int64, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
		}
		grouped[accID] = ids
	}
	return grouped
}

func processAdvisoryBatch(ctx context.Context, grouped map[int][]int64) error {
	for rhAccountID, advisoryIDs := range grouped {
		utils.LogInfo("rh_account_id", rhAccountID, "advisory_count", len(advisoryIDs), "refreshing account advisory caches")
		err := database.DB(ctx).Exec("SELECT refresh_account_advisory_caches_multi(?, ?)", pq.Array(advisoryIDs), rhAccountID).Error //nolint:lll
		if err != nil {
			utils.LogError("err", err, "rh_account_id", rhAccountID, "failed to refresh account advisory caches")
			return base.WrapFatalDBError(err, "refresh account advisory caches")
		}

		CheckAdvisoryDrift(rhAccountID, advisoryIDs)

		if err := publishNewAdvisoryNotification(rhAccountID, advisoryIDs); err != nil {
			utils.LogError("err", err, "rh_account_id", rhAccountID, "failed to publish new advisory notification")
			return base.WrapFatalKafkaError(err, "publish new advisory notification")
		}
	}
	return nil
}
