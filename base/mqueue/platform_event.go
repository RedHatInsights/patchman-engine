package mqueue

import (
	"app/base"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type PlatformEvent struct {
	ID          string                 `json:"id"`
	Type        *string                `json:"type"`
	Timestamp   *base.Rfc3339Timestamp `json:"timestamp"`
	Account     *string                `json:"account"`
	AccountID   int                    `json:"account_id"`
	OrgID       *string                `json:"org_id,omitempty"`
	B64Identity *string                `json:"b64_identity"`
	URL         *string                `json:"url"`
	SystemIDs   []string               `json:"system_ids,omitempty"`
	RequestIDs  []string               `json:"request_ids,omitempty"`
}

type InventoryAID struct {
	InventoryID string
	RhAccountID int
}

type AccountInfo struct {
	AccountName *string
	OrgID       *string
}

type EvalData struct {
	RhAccountID int
	InventoryID string
	RequestID   string
	AccountInfo AccountInfo
}

type PlatformEvents []PlatformEvent
type InventoryAIDs []InventoryAID
type EvalDataSlice []EvalData

type groupedData map[int][]string
type groupedAccountInfo map[int]AccountInfo

func (event *PlatformEvent) createKafkaMessage() (KafkaMessage, error) {
	data, err := json.Marshal(event) //nolint:gosec
	if err != nil {
		return KafkaMessage{}, errors.Wrap(err, "Serializing event")
	}
	return KafkaMessage{Value: data}, err
}

func (event *PlatformEvent) GetAccountName() string {
	if event.Account == nil {
		return ""
	}
	return *event.Account
}

func (event *PlatformEvent) GetOrgID() string {
	if event.OrgID == nil {
		return ""
	}
	return *event.OrgID
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

func (data *InventoryAIDs) groupData() (int, *groupedData) {
	// group systems by account
	grouped := groupedData{}
	for _, aid := range *data {
		grouped[aid.RhAccountID] = append(grouped[aid.RhAccountID], aid.InventoryID)
	}
	return batchSize(grouped), &grouped
}

func (data *EvalDataSlice) groupData() (int, *groupedData, *groupedData, *groupedAccountInfo) {
	// group systems by account
	groupedSys := groupedData{}
	groupedReqID := groupedData{}
	accountInfo := groupedAccountInfo{}
	for _, d := range *data {
		groupedSys[d.RhAccountID] = append(groupedSys[d.RhAccountID], d.InventoryID)
		groupedReqID[d.RhAccountID] = append(groupedReqID[d.RhAccountID], d.RequestID)
		if _, has := accountInfo[d.RhAccountID]; !has {
			accountInfo[d.RhAccountID] = d.AccountInfo
		}
	}
	return batchSize(groupedSys), &groupedSys, &groupedReqID, &accountInfo
}

func (data *InventoryAIDs) WriteEvents(ctx context.Context, w Writer) error {
	batches, groupedSys := data.groupData()
	// create events, per BatchSize of systems from one account
	now := base.Rfc3339Timestamp(time.Now())
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
			})
		}
	}
	// write events to queue
	err := writePlatformEvents(ctx, w, events...)
	return err
}

func (data *EvalDataSlice) WriteEvents(ctx context.Context, w Writer) error {
	batches, groupedSys, groupedReqID, accountInfo := data.groupData()
	groupedSysVal := groupedData{}
	groupedReqIDVal := groupedData{}
	if groupedSys != nil {
		groupedSysVal = *groupedSys
	}
	if groupedReqID != nil {
		groupedReqIDVal = *groupedReqID
	}
	// create events, per BatchSize of systems from one account
	now := base.Rfc3339Timestamp(time.Now())
	events := make(PlatformEvents, 0, batches)
	for acc, accDetails := range *accountInfo {
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
					Account:    accDetails.AccountName,
					OrgID:      accDetails.OrgID,
					RequestIDs: groupedReqIDVal[acc][start:end],
				},
			)
		}
	}
	// write events to queue
	err := writePlatformEvents(ctx, w, events...)
	return err
}
