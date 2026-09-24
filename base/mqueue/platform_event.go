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

func batchSize(grouped map[int][]uuid.UUID) int {
	// compute how many batches we will create
	var batches = 0
	for _, ev := range grouped {
		batches += (len(ev) + BatchSize - 1) / BatchSize
	}
	return batches
}

func (evals EvalDataSlice) getAccountEvalData() (
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
	return batchSize(invs), invs, reqs, tps, orgs
}

func (evals EvalDataSlice) WriteEvents(ctx context.Context, w Writer) error {
	batches, accInvs, reqs, tps, orgs := evals.getAccountEvalData()
	now := types.Rfc3339Timestamp(time.Now())
	events := make(PlatformEvents, 0, batches)
	for acc, invs := range accInvs {
		for start := 0; start < len(invs); start += BatchSize {
			end := start + BatchSize
			if end > len(invs) {
				end = len(invs)
			}
			events = append(events, PlatformEvent{
				Timestamp:    &now,
				AccountID:    acc,
				SystemIDs:    invs[start:end],
				RequestIDs:   reqs[acc][start:end],
				Traceparents: tps[acc][start:end],
				OrgID:        orgs[acc],
			})
		}
	}
	// write events to queue
	return writePlatformEvents(ctx, w, events...)
}
