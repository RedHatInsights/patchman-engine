package mqueue

import (
	"app/base/utils"
	"encoding/json"
	"github.com/segmentio/kafka-go"
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

func (t *Reader) HandleEvents(handler EventHandler) {
	t.HandleMessages(makeKafkaHandler(handler))
}
