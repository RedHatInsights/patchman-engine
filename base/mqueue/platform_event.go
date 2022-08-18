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

type InventoryAID struct {
	InventoryID string
	RhAccountID int
	OrgID       *string
}

type EvalData struct {
	RhAccountID int
	InventoryID string
	RequestID   string
	OrgID       *string
}

type PlatformEvents []PlatformEvent
type InventoryAIDs []InventoryAID
type EvalDataSlice []EvalData

type groupedData map[int][]string
type orgIDMap map[int]*string

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

func batchSize(grouped groupedData) int {
	// compute how many batches we will create
	var batches = 0
	for _, ev := range grouped {
		batches += len(ev)/BatchSize + 1
	}
	return batches
}

func (data *InventoryAIDs) groupData() (int, *groupedData, *orgIDMap) {
	// group systems by account
	grouped := groupedData{}
	orgIDs := orgIDMap{}
	for _, aid := range *data {
		grouped[aid.RhAccountID] = append(grouped[aid.RhAccountID], aid.InventoryID)
		if _, has := orgIDs[aid.RhAccountID]; !has {
			orgIDs[aid.RhAccountID] = aid.OrgID
		}
	}
	return batchSize(grouped), &grouped, &orgIDs
}

func (data *EvalDataSlice) groupData() (int, *groupedData, *groupedData, *orgIDMap) {
	// group systems by account
	groupedSys := groupedData{}
	groupedReqID := groupedData{}
	orgIDMap := orgIDMap{}
	// accountInfo := groupedAccountInfo{}
	for _, d := range *data {
		groupedSys[d.RhAccountID] = append(groupedSys[d.RhAccountID], d.InventoryID)
		groupedReqID[d.RhAccountID] = append(groupedReqID[d.RhAccountID], d.RequestID)
		if _, has := orgIDMap[d.RhAccountID]; !has {
			orgIDMap[d.RhAccountID] = d.OrgID
		}
	}
	return batchSize(groupedSys), &groupedSys, &groupedReqID, &orgIDMap
}

func (data *InventoryAIDs) WriteEvents(ctx context.Context, w Writer) error {
	batches, groupedSys, orgIDs := data.groupData()
	// create events, per BatchSize of systems from one account
	now := types.Rfc3339Timestamp(time.Now())
	events := make(PlatformEvents, 0, batches)
	for acc, ev := range *groupedSys {
		for start := 0; start < len(ev); start += BatchSize {
			end := start + BatchSize
			if end > len(ev) {
				end = len(ev)
			}
			events = append(events, PlatformEvent{
				Timestamp: &now,
				AccountID: acc,
				SystemIDs: ev[start:end],
				OrgID:     (*orgIDs)[acc],
			})
		}
	}
	// write events to queue
	err := writePlatformEvents(ctx, w, events...)
	return err
}

func (data *EvalDataSlice) WriteEvents(ctx context.Context, w Writer) error {
	batches, groupedSys, groupedReqID, orgIDMap := data.groupData()
	groupedSysVal := groupedData{}
	groupedReqIDVal := groupedData{}
	if groupedSys != nil {
		groupedSysVal = *groupedSys
	}
	if groupedReqID != nil {
		groupedReqIDVal = *groupedReqID
	}
	// create events, per BatchSize of systems from one account
	now := types.Rfc3339Timestamp(time.Now())
	events := make(PlatformEvents, 0, batches)
	for acc, orgID := range *orgIDMap {
		nEvents := len(groupedSysVal[acc])
		for start := 0; start < nEvents; start += BatchSize {
			end := start + BatchSize
			if end > nEvents {
				end = nEvents
			}
			events = append(
				events,
				PlatformEvent{
					Timestamp:  &now,
					AccountID:  acc,
					SystemIDs:  groupedSysVal[acc][start:end],
					OrgID:      orgID,
					RequestIDs: groupedReqIDVal[acc][start:end],
				},
			)
		}
	}
	// write events to queue
	err := writePlatformEvents(ctx, w, events...)
	return err
}
