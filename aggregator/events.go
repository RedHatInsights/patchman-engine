package aggregator

import (
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"sync"
	"time"

	"github.com/lib/pq"
)

var (
	batchSize      int
	flushTimeout   time.Duration
	advisoryBuffer []mqueue.AdvisoryUpdateEvent
	bufferLock     sync.Mutex
	flushTimer     *time.Timer
)

func initBuffer() {
	advisoryBuffer = make([]mqueue.AdvisoryUpdateEvent, 0, batchSize+1)
	flushTimer = time.AfterFunc(87600*time.Hour, func() {
		utils.LogInfo("flushing advisory buffer after timeout")
		flushAdvisoryBuffer()
	})
}

func flushAdvisoryBuffer() {
	bufferLock.Lock()
	if len(advisoryBuffer) == 0 {
		bufferLock.Unlock()
		return
	}
	// Copy and unlock before processing so new events can accumulate while DB work runs
	batch := make([]mqueue.AdvisoryUpdateEvent, len(advisoryBuffer))
	copy(batch, advisoryBuffer)
	advisoryBuffer = advisoryBuffer[:0]
	bufferLock.Unlock()

	grouped := groupAdvisoryUpdates(batch)
	processAdvisoryBatch(grouped)
}

func advisoryUpdateHandler(event mqueue.AdvisoryUpdateEvent) error {
	bufferLock.Lock()
	advisoryBuffer = append(advisoryBuffer, event)
	flushTimer.Reset(flushTimeout)
	shouldFlush := len(advisoryBuffer) >= batchSize
	bufferLock.Unlock()

	if shouldFlush {
		utils.LogInfo("flushing full advisory buffer")
		flushAdvisoryBuffer()
	}
	return nil
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

func processAdvisoryBatch(grouped map[int][]int64) {
	for rhAccountID, advisoryIDs := range grouped {
		utils.LogInfo("rh_account_id", rhAccountID, "advisory_count", len(advisoryIDs), "refreshing account advisory caches")
		err := database.DB.Exec("SELECT refresh_account_advisory_caches_multi(?, ?)", pq.Array(advisoryIDs), rhAccountID).Error //nolint:lll
		if err != nil {
			utils.LogError("err", err, "rh_account_id", rhAccountID, "failed to refresh account advisory caches")
			continue
		}

		checkAdvisoryDrift(rhAccountID, advisoryIDs)

		if err := publishNewAdvisoryNotification(rhAccountID, advisoryIDs); err != nil {
			utils.LogError("err", err, "rh_account_id", rhAccountID, "failed to publish new advisory notification")
		}
	}
}
