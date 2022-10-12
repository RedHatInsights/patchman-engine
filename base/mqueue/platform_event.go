package mqueue

import (
	"app/base/types"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type PlatformEvent struct {
	ID          string                  `json:"id"`
	Type        *string                 `json:"type"`
	Timestamp   *types.Rfc3339Timestamp `json:"timestamp"`
	AccountID   int                     `json:"account_id"`
	OrgID       *string                 `json:"org_id,omitempty"`
	B64Identity *string                 `json:"b64_identity"`
	URL         *string                 `json:"url"`
	SystemIDs   []string                `json:"system_ids,omitempty"`
	RequestIDs  []string                `json:"request_ids,omitempty"`
}

type EvalData struct {
	InventoryID string
	RhAccountID int
	RequestID   string
	OrgID       *string
}

type PlatformEvents []PlatformEvent
type EvalDataSlice []EvalData

type accountInventories map[int][]string
type accountRequests map[int][]string
type orgIDs map[int]*string

func (event *PlatformEvent) createKafkaMessage() (KafkaMessage, error) {
	data, err := json.Marshal(event) //nolint:gosec
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

func batchSize(grouped map[int][]string) int {
	// compute how many batches we will create
	var batches = 0
	for _, ev := range grouped {
		batches += len(ev)/BatchSize + 1
	}
	return batches
}

func (evals EvalDataSlice) getAccountEvalData() (int, accountInventories, accountRequests, orgIDs) {
	// group systems by account
	invs := accountInventories{}
	reqs := accountRequests{}
	orgs := orgIDs{}
	for _, e := range evals {
		invs[e.RhAccountID] = append(invs[e.RhAccountID], e.InventoryID)
		reqs[e.RhAccountID] = append(reqs[e.RhAccountID], e.RequestID)
		if _, has := orgs[e.RhAccountID]; !has {
			orgs[e.RhAccountID] = e.OrgID
		}
	}
	return batchSize(invs), invs, reqs, orgs
}

func (evals EvalDataSlice) WriteEvents(ctx context.Context, w Writer) error {
	batches, accInvs, reqs, orgs := evals.getAccountEvalData()
	// create events, per BatchSize of systems from one account
	now := types.Rfc3339Timestamp(time.Now())
	events := make(PlatformEvents, 0, batches)
	for acc, invs := range accInvs {
		for start := 0; start < len(invs); start += BatchSize {
			end := start + BatchSize
			if end > len(invs) {
				end = len(invs)
			}
			events = append(events, PlatformEvent{
				Timestamp:  &now,
				AccountID:  acc,
				SystemIDs:  invs[start:end],
				RequestIDs: reqs[acc][start:end],
				OrgID:      orgs[acc],
			})
		}
	}
	// write events to queue
	err := writePlatformEvents(ctx, w, events...)
	return err
}
