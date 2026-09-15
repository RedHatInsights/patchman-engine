// Provides some basic methods for interacting with platform kafka message queue
package mqueue

import (
	"app/base"
	"app/base/utils"
	"context"
	"errors"
	format "fmt"
	"io"
	"strings"
	"sync"

	"github.com/lestrrat-go/backoff/v2"
	"github.com/segmentio/kafka-go"
)

const errContextCanceled = "context canceled"

// By wrapping raw value we can add new methods & ensure methods of wrapped type are callable
type Reader interface {
	HandleMessages(ctx context.Context, handler MessageHandler)
	io.Closer
}

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...KafkaMessage) error
}

func createLoggerFunc(counter Counter) func(fmt string, args ...interface{}) {
	if counter == nil {
		panic("kafka error counter nil")
	}

	fn := func(fmt string, args ...interface{}) {
		counter.Inc()
		msg := format.Sprintf(fmt, args...)
		if strings.Contains(msg, "failed to dial") ||
			strings.Contains(msg, "unknown error reading partition") {
			utils.LogWarn("type", "kafka", msg)
		} else {
			utils.LogError("type", "kafka", msg)
		}
		if strings.Contains(msg, "Group Load In Progress") {
			utils.LogPanic("Kafka client stuck detected!!!")
		}
	}
	return fn
}

type KafkaMessage struct {
	Key     []byte
	Value   []byte
	Headers []kafka.Header
}

type MessageHandler func(ctx context.Context, message KafkaMessage) error

func MakeRetryingHandler(handler MessageHandler) MessageHandler {
	return func(ctx context.Context, message KafkaMessage) error {
		var err error
		var attempt int

		backoffCtx, cancel := context.WithCancel(context.Background())
		backoffState := policy.Start(backoffCtx)
		defer cancel()
		for backoff.Continue(backoffState) {
			if err = handler(ctx, message); err == nil || !errors.Is(err, base.ErrFatal) {
				return nil
			}
			utils.LogError("err", err, "attempt", attempt, "Try failed")
			attempt++
		}
		if err != nil && errors.Is(err, base.ErrFatal) {
			return err
		}
		return nil
	}
}

type CreateReader func(topic string) Reader
type CreateWriter func(topic string) Writer

func runReader(ctx context.Context, wg *sync.WaitGroup, topic string, createReader CreateReader,
	msgHandler MessageHandler) {
	defer wg.Done()
	defer utils.LogPanics(true)
	reader := createReader(topic)
	defer reader.Close()
	reader.HandleMessages(ctx, msgHandler)
}

func SpawnReader(ctx context.Context, wg *sync.WaitGroup, topic string, createReader CreateReader,
	msgHandler MessageHandler) {
	wg.Add(1)
	go runReader(ctx, wg, topic, createReader, msgHandler)
}
