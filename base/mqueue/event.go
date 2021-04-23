package mqueue

import (
	"app/base"
	"app/base/utils"
	"encoding/json"
	"github.com/lestrrat-go/backoff"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"golang.org/x/net/context"
	"time"
)

const BatchSize = 4000

var policy = backoff.NewExponential(
	backoff.WithInterval(time.Second),
	backoff.WithMaxRetries(5),
)

type PlatformEvent struct {
	ID          string                 `json:"id"`
	Type        *string                `json:"type"`
	Timestamp   *base.Rfc3339Timestamp `json:"timestamp"`
	Account     *string                `json:"account"`
	AccountID   int                    `json:"account_id"`
	B64Identity *string                `json:"b64_identity"`
	URL         *string                `json:"url"`
	SystemIDs   []string               `json:"system_ids,omitempty"`
}

type InventoryAID struct {
	InventoryID string
	RhAccountID int
}

type EventHandler func(message PlatformEvent) error

// Performs parsing of kafka message, and then dispatches this message into provided functions
func MakeMessageHandler(eventHandler EventHandler) MessageHandler {
	return func(m kafka.Message) error {
		var event PlatformEvent
		err := json.Unmarshal(m.Value, &event)
		// Not a fatal error, invalid data format, log and skip
		if err != nil {
			utils.Log("err", err.Error()).Error("Could not deserialize platform event")
			return nil
		}
		return eventHandler(event)
	}
}

// nolint: scopelint
func WriteEvents(ctx context.Context, w Writer, events ...PlatformEvent) error {
	msgs := make([]kafka.Message, len(events))
	for i, ev := range events {
		data, err := json.Marshal(&ev) //nolint:gosec
		if err != nil {
			return errors.Wrap(err, "Serializing event")
		}
		msgs[i] = kafka.Message{Value: data}
	}
	return w.WriteMessages(ctx, msgs...)
}

func SendMessages(ctx context.Context, w Writer, inventoryAIDs ...InventoryAID) error {
	// group systems by account
	grouped := map[int][]string{}
	for _, aid := range inventoryAIDs {
		grouped[aid.RhAccountID] = append(grouped[aid.RhAccountID], aid.InventoryID)
	}

	// compute how many batches we will create
	var batches int = 0
	for _, ev := range grouped {
		batches += len(ev)/BatchSize + 1
	}

	// create events, per BatchSize of systems from one account
	now := base.Rfc3339Timestamp(time.Now())
	events := make([]PlatformEvent, 0, batches)
	for acc, ev := range grouped {
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
	err := WriteEvents(ctx, w, events...)
	return err
}
