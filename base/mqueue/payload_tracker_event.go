package mqueue

import (
	"app/base"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type PayloadTrackerEvent struct {
	Service     string                      `json:"service"`
	Account     *string                     `json:"account,omitempty"`
	OrgID       *string                     `json:"org_id,omitempty"`
	RequestID   *string                     `json:"request_id"`
	InventoryID string                      `json:"inventory_id"`
	Status      string                      `json:"status"`
	StatusMsg   string                      `json:"status_msg,omitempty"`
	Date        *base.Rfc3339TimestampWithZ `json:"date"`
}

type PayloadTrackerEvents []PayloadTrackerEvent

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

func writePayloadTrackerEvents(ctx context.Context, w Writer, events ...PayloadTrackerEvent) error {
	var err error
	now := base.Rfc3339TimestampWithZ(time.Now())
	for _, ev := range events {
		if ev.RequestID != nil && (ev.Account != nil || ev.OrgID != nil) {
			// Send only messages from listener and evaluator-upload
			ev.Service = "patchman"
			ev.Date = &now
			err = ev.write(ctx, w)
		}
	}
	return err
}

func (events *PayloadTrackerEvents) WriteEvents(ctx context.Context, w Writer) error {
	return writePayloadTrackerEvents(ctx, w, *events...)
}

func (event *PayloadTrackerEvent) WriteEvents(ctx context.Context, w Writer) error {
	return writePayloadTrackerEvents(ctx, w, *event)
}
