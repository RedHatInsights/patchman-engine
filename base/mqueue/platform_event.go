package mqueue

import (
	"app/base/types"
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// PlatformEvent ID is typed as uuid.UUID to match the inventory service contract:
// https://github.com/RedHatInsights/insights-host-inventory/blob/master/swagger/host_events.spec.yaml
type PlatformEvent struct {
	ID           uuid.UUID               `json:"id"`
	Type         *string                 `json:"type"`
	Timestamp    *types.Rfc3339Timestamp `json:"timestamp"`
	AccountID    int                     `json:"account_id"`
	OrgID        *string                 `json:"org_id,omitempty"`
	B64Identity  *string                 `json:"b64_identity"`
	URL          *string                 `json:"url"`
	SystemIDs    []uuid.UUID             `json:"system_ids,omitempty"`
	RequestIDs   []string                `json:"request_ids,omitempty"`
	Traceparents []string                `json:"traceparents,omitempty"`
	// SkipNotifications suppresses instant advisory notification publish for this event.
	// Evaluator still marks matching advisory_account_data.notified so later evals do not flood.
	// Used by recovery recalc; omit/false for normal upload/recalc traffic.
	SkipNotifications bool `json:"skip_notifications,omitempty"`
}

type EvalData struct {
	InventoryID uuid.UUID
	RhAccountID int
	RequestID   string
	OrgID       *string
	Traceparent string
}

type PlatformEvents []PlatformEvent
type EvalDataSlice []EvalData

type accountInventories map[int][]uuid.UUID
type accountRequests map[int][]string
type accountTraceparents map[int][]string
type orgIDs map[int]*string

func (event *PlatformEvent) createKafkaMessage() (KafkaMessage, error) {
	data, err := sonic.Marshal(event)
	if err != nil {
		return KafkaMessage{}, errors.Wrap(err, "Serializing event")
	}
	return KafkaMessage{Value: data}, err
}

func (event *PlatformEvent) GetOrgID() string {
	if event.OrgID == nil {
		return ""
	}
	return *event.OrgID
}

func (event *PlatformEvent) GetURL() string {
	if event.URL == nil {
		return ""
	}
	return *event.URL
}

func writePlatformEvents(ctx context.Context, w Writer, events ...PlatformEvent) error {
	var err error
	msgs := make([]KafkaMessage, len(events))
	for i, ev := range events {
		msgs[i], err = ev.createKafkaMessage()
		if err != nil {
			return err
		}
	}
	return w.WriteMessages(ctx, msgs...)
}

func batchCount(grouped map[int][]uuid.UUID, size int) int {
	if size <= 0 {
		size = BatchSize
	}
	var batches = 0
	for _, ev := range grouped {
		batches += (len(ev) + size - 1) / size
	}
	return batches
}

func (evals EvalDataSlice) getAccountEvalData(size int) (
	int, accountInventories, accountRequests, accountTraceparents, orgIDs) {
	// group systems by account
	invs := accountInventories{}
	reqs := accountRequests{}
	tps := accountTraceparents{}
	orgs := orgIDs{}
	for _, e := range evals {
		invs[e.RhAccountID] = append(invs[e.RhAccountID], e.InventoryID)
		reqs[e.RhAccountID] = append(reqs[e.RhAccountID], e.RequestID)
		tps[e.RhAccountID] = append(tps[e.RhAccountID], e.Traceparent)
		if _, has := orgs[e.RhAccountID]; !has {
			orgs[e.RhAccountID] = e.OrgID
		}
	}
	return batchCount(invs, size), invs, reqs, tps, orgs
}

func (evals EvalDataSlice) WriteEvents(ctx context.Context, w Writer) error {
	return evals.writeEvents(ctx, w, BatchSize, false)
}

// WriteEventsSkipNotifications publishes recalc events in batches of batchSize systems
// per account with SkipNotifications set. Used by one-off recovery jobs.
func (evals EvalDataSlice) WriteEventsSkipNotifications(ctx context.Context, w Writer, batchSize int) error {
	return evals.writeEvents(ctx, w, batchSize, true)
}

func (evals EvalDataSlice) writeEvents(ctx context.Context, w Writer, size int, skipNotifications bool) error {
	if size <= 0 {
		size = BatchSize
	}
	batches, accInvs, reqs, tps, orgs := evals.getAccountEvalData(size)
	now := types.Rfc3339Timestamp(time.Now())
	events := make(PlatformEvents, 0, batches)
	for acc, invs := range accInvs {
		for start := 0; start < len(invs); start += size {
			end := start + size
			if end > len(invs) {
				end = len(invs)
			}
			events = append(events, PlatformEvent{
				Timestamp:         &now,
				AccountID:         acc,
				SystemIDs:         invs[start:end],
				RequestIDs:        reqs[acc][start:end],
				Traceparents:      tps[acc][start:end],
				OrgID:             orgs[acc],
				SkipNotifications: skipNotifications,
			})
		}
	}
	return writePlatformEvents(ctx, w, events...)
}
