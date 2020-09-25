package mqueue

import (
	"app/base/utils"
	"encoding/json"
	"github.com/lestrrat-go/backoff"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"golang.org/x/net/context"
	"time"
)

var policy = backoff.NewExponential(
	backoff.WithInterval(time.Second),
	backoff.WithMaxRetries(5),
)

type PlatformEvent struct {
	ID          string  `json:"id"`
	Type        *string `json:"type"`
	Timestamp   *string `json:"timestamp"`
	Account     *string `json:"account"`
	B64Identity *string `json:"b64_identity"`
	URL         *string `json:"url"`
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
