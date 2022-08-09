package mqueue

import (
	"app/base/types"
	"app/base/utils"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type PayloadTrackerEvent struct {
	Service     string                       `json:"service"`
	Account     *string                      `json:"account,omitempty"`
	OrgID       *string                      `json:"org_id,omitempty"`
	RequestID   *string                      `json:"request_id"`
	InventoryID string                       `json:"inventory_id"`
	Status      string                       `json:"status"`
	StatusMsg   string                       `json:"status_msg,omitempty"`
	Date        *types.Rfc3339TimestampWithZ `json:"date"`
}

type PayloadTrackerEvents []PayloadTrackerEvent

var enablePayloadTracker = utils.GetBoolEnvOrDefault("ENABLE_PAYLOAD_TRACKER", true)

func (event *PayloadTrackerEvent) write(ctx context.Context, w Writer) error {
	data, err := json.Marshal(event) //nolint:gosec
	if err != nil {
		return errors.Wrap(err, "Serializing event")
	}
	msg := KafkaMessage{Value: data}
	if err != nil {
		return err
	}
	return w.WriteMessages(ctx, msg)
}

func writeEvent(ctx context.Context, w Writer, event *PayloadTrackerEvent,
	timestamp *types.Rfc3339TimestampWithZ) (err error) {
	if event.RequestID != nil && (event.Account != nil || event.OrgID != nil) {
		// Send only messages from listener and evaluator-upload
		event.Service = "patchman"
		event.Date = timestamp
		err = event.write(ctx, w)
	}
	return err
}

func (events *PayloadTrackerEvents) WriteEvents(ctx context.Context, w Writer) error {
	if !enablePayloadTracker {
		return nil
	}
	var err error
	now := types.Rfc3339TimestampWithZ(time.Now())
	for _, event := range *events {
		event := event // necessary, G601: Implicit memory aliasing in for loop. (gosec)
		err = writeEvent(ctx, w, &event, &now)
	}
	return err
}

func (event *PayloadTrackerEvent) WriteEvents(ctx context.Context, w Writer) error {
	if !enablePayloadTracker {
		return nil
	}
	now := types.Rfc3339TimestampWithZ(time.Now())
	err := writeEvent(ctx, w, event, &now)
	return err
}
