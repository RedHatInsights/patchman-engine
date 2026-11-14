package aggregator

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"bytes"
	"context"
	"errors"
	"sort"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	batchSize    int
	flushTimeout time.Duration
)

const advisoryCountSQL = `
SELECT count(sa.*) FILTER (WHERE sa.status_id = 0) AS systems_installable,
       count(sa.*) AS systems_applicable
  FROM system_advisories sa
  JOIN system_inventory si
    ON sa.rh_account_id = si.rh_account_id AND sa.system_id = si.id
 WHERE si.stale = FALSE
   AND sa.rh_account_id = ?
   AND sa.advisory_id = ?
   AND si.rh_account_id = ?
   AND si.workspace_id = ?
 GROUP BY sa.advisory_id, sa.rh_account_id, si.workspace_id`

type advisoryRecountCount struct {
	SystemsInstallable int
	SystemsApplicable  int
}

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

type advisoryRecountKey struct {
	RhAccountID int
	WorkspaceID uuid.UUID
	AdvisoryID  int64
}

type advisoryRecount struct {
	advisoryRecountKey
	ProducedAt time.Time
}

func groupAdvisoryUpdates(events []mqueue.AdvisoryUpdateEvent) []advisoryRecount {
	newest := make(map[advisoryRecountKey]time.Time)
	for i := range events {
		event := &events[i]
		producedAt := time.Time{}
		if ts := event.ProducedAt.Time(); ts != nil {
			producedAt = *ts
		}
		for _, advisoryID := range event.AdvisoryIDs {
			key := advisoryRecountKey{event.RhAccountID, event.WorkspaceID, advisoryID}
			if existing, ok := newest[key]; !ok || producedAt.After(existing) {
				newest[key] = producedAt
			}
		}
	}

	requests := make([]advisoryRecount, 0, len(newest))
	for key, producedAt := range newest {
		requests = append(requests, advisoryRecount{
			RhAccountID: key.RhAccountID,
			WorkspaceID: key.WorkspaceID,
			AdvisoryID:  key.AdvisoryID,
			ProducedAt:  producedAt,
		})
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].RhAccountID != requests[j].RhAccountID {
			return requests[i].RhAccountID < requests[j].RhAccountID
		}
		if c := bytes.Compare(requests[i].WorkspaceID[:], requests[j].WorkspaceID[:]); c != 0 {
			return c < 0
		}
		return requests[i].AdvisoryID < requests[j].AdvisoryID
	})
	return requests
}

func processAdvisoryBatch(ctx context.Context, requests []advisoryRecount) error {
	utils.LogInfo("advisory_count", len(requests), "refreshing account advisory caches")

	accountAdvisories := make(map[int][]int64)
	for _, req := range requests {
		err := recountAdvisory(ctx, req.RhAccountID, req.WorkspaceID, req.AdvisoryID, req.ProducedAt)
		if err != nil {
			utils.LogError("err", err, "rh_account_id", req.RhAccountID,
				"failed to refresh account advisory caches")
			return base.WrapFatalDBError(err, "refresh account advisory caches")
		}
		accountAdvisories[req.RhAccountID] = append(accountAdvisories[req.RhAccountID], req.AdvisoryID)
	}

	for rhAccountID, advisoryIDs := range accountAdvisories {
		if err := publishNewAdvisoryNotification(rhAccountID, advisoryIDs); err != nil {
			utils.LogError("err", err, "rh_account_id", rhAccountID, "failed to publish new advisory notification")
			return base.WrapFatalKafkaError(err, "publish new advisory notification")
		}
	}
	return nil
}

func recountAdvisory(ctx context.Context, rhAccountID int, workspaceID uuid.UUID, advisoryID int64,
	producedAt time.Time) error {
	return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.AccountAdvisory
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("rh_account_id = ? AND workspace_id = ? AND advisory_id = ?", rhAccountID, workspaceID, advisoryID).
			Take(&row).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && row.Updated.After(producedAt) {
			utils.LogWarn(
				"rh_account_id", rhAccountID,
				"workspace_id", workspaceID,
				"advisory_id", advisoryID,
				"updated", row.Updated,
				"produced_at", producedAt,
				"skipping advisory recount, updated is younger than produced_at",
			)
			return nil
		}

		var counts []advisoryRecountCount
		err = tx.Raw(advisoryCountSQL, rhAccountID, advisoryID, rhAccountID, workspaceID).Scan(&counts).Error
		if err != nil {
			return err
		}
		if len(counts) == 0 {
			return tx.Where("rh_account_id = ? AND workspace_id = ? AND advisory_id = ?",
				rhAccountID, workspaceID, advisoryID).
				Delete(&models.AccountAdvisory{}).Error
		}

		rec := models.AccountAdvisory{
			AdvisoryID:         advisoryID,
			RhAccountID:        rhAccountID,
			WorkspaceID:        workspaceID,
			SystemsApplicable:  counts[0].SystemsApplicable,
			SystemsInstallable: counts[0].SystemsInstallable,
			Updated:            time.Now(),
		}
		return database.OnConflictUpdateMulti(tx,
			[]string{"rh_account_id", "workspace_id", "advisory_id"},
			"systems_installable", "systems_applicable", "updated").
			Create(&rec).Error
	})
}
