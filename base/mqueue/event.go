package mqueue

import (
	"app/base/utils"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"golang.org/x/net/context"
)

type PlatformEvent struct {
	ID string `json:"id"`

	Type        *string `json:"type"`
	Timestamp   *string `json:"timestamp"`
	Account     *string `json:"account"`
	B64Identity *string `json:"b64_identity"`
	URL         *string `json:"url"`
}

type EventHandler func(event PlatformEvent)

// Performs parsing of kafka message, and then dispatches this message into provided functions
func makeKafkaHandler(eventHandler EventHandler) KafkaHandler {
	return func(m kafka.Message) {
		var event PlatformEvent
		err := json.Unmarshal(m.Value, &event)
		if err != nil {
			utils.Log("err", err.Error()).Error("Could not deserialize platform event")
			return
		}
		eventHandler(event)
	}
}

func (t *readerImpl) HandleEvents(handler EventHandler) {
	t.HandleMessages(makeKafkaHandler(handler))
}

// nolint: scopelint
func (t *writerImpl) WriteEvents(ctx context.Context, events ...PlatformEvent) error {
	msgs := make([]kafka.Message, len(events))
	for i, ev := range events {
		data, err := json.Marshal(&ev)
		if err != nil {
			return errors.Wrap(err, "Serializing event")
		}
		msgs[i] = kafka.Message{Value: data}
	}
	return t.WriteMessages(ctx, msgs...)
}
