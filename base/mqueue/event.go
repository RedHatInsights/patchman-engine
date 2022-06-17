package mqueue

import (
	"app/base/utils"
	"encoding/json"
	"time"

	"github.com/lestrrat-go/backoff"
	"golang.org/x/net/context"
)

var BatchSize = utils.GetIntEnvOrDefault("MSG_BATCH_SIZE", 4000)

var policy = backoff.NewExponential(
	backoff.WithInterval(time.Second),
	backoff.WithMaxRetries(5),
)

type EventHandler func(message PlatformEvent) error

type MessageData interface {
	WriteEvents(ctx context.Context, w Writer) error
}

// Performs parsing of kafka message, and then dispatches this message into provided functions
func MakeMessageHandler(eventHandler EventHandler) MessageHandler {
	return func(m KafkaMessage) error {
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

func SendMessages(ctx context.Context, w Writer, data MessageData) error {
	return data.WriteEvents(ctx, w)
}
