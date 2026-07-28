package mqueue

import (
	"app/base/utils"
	"context"
	"time"

	"github.com/lestrrat-go/backoff/v2"
)

var BatchSize = utils.PodConfig.GetInt("msg_batch_size", 4000)

var policy = backoff.Exponential(
	backoff.WithMinInterval(time.Second),
	backoff.WithMaxRetries(5),
)

type MessageData interface {
	WriteEvents(ctx context.Context, w Writer) error
}

func SendMessages(ctx context.Context, w Writer, data MessageData) error {
	return data.WriteEvents(ctx, w)
}
