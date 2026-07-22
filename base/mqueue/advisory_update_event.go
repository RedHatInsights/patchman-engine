package mqueue

import (
	"app/base/types"
	"app/base/utils"
	"context"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type AdvisoryUpdateEventHandler func(event AdvisoryUpdateEvent) error

func MakeAdvisoryUpdateHandler(handler AdvisoryUpdateEventHandler) MessageHandler {
	return func(m KafkaMessage) error {
		var event AdvisoryUpdateEvent
		err := sonic.Unmarshal(m.Value, &event)
		if err != nil {
			utils.LogError("err", err, "Could not deserialize advisory update event")
			return nil
		}
		return handler(event)
	}
}

type AdvisoryUpdateEvent struct {
	RhAccountID int                    `json:"rh_account_id"`
	WorkspaceID uuid.UUID              `json:"workspace_id"`
	AdvisoryIDs []int64                `json:"advisory_ids"`
	ProducedAt  types.Rfc3339Timestamp `json:"produced_at"`
}

type AdvisoryUpdateEvents []AdvisoryUpdateEvent

func (event *AdvisoryUpdateEvent) createKafkaMessage() (KafkaMessage, error) {
	data, err := sonic.Marshal(event)
	if err != nil {
		return KafkaMessage{}, errors.Wrap(err, "Serializing advisory update event")
	}
	return KafkaMessage{Value: data}, nil
}

func (events AdvisoryUpdateEvents) WriteEvents(ctx context.Context, w Writer) error {
	msgs := make([]KafkaMessage, 0, len(events))
	for i := range events {
		msg, err := events[i].createKafkaMessage()
		if err != nil {
			return err
		}
		msgs = append(msgs, msg)
	}
	return w.WriteMessages(ctx, msgs...)
}
