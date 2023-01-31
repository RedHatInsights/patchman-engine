// Provides some basic methods for interacting with platform kafka message queue
package mqueue

import (
	"app/base"
	"app/base/utils"
	"context"
	format "fmt"
	"io"
	"strings"
	"sync"

	"github.com/lestrrat-go/backoff"
)

const errContextCanceled = "context canceled"

// By wrapping raw value we can add new methods & ensure methods of wrapped type are callable
type Reader interface {
	HandleMessages(handler MessageHandler)
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
		utils.LogError("type", "kafka", format.Sprintf(fmt, args...))
		if strings.Contains(fmt, "Group Load In Progress") {
			utils.LogPanic("Kafka client stuck detected!!!")
		}
	}
	return fn
}

type KafkaMessage struct {
	Key   []byte
	Value []byte
}

type MessageHandler func(message KafkaMessage) error

func MakeRetryingHandler(handler MessageHandler) MessageHandler {
	return func(message KafkaMessage) error {
		var err error
		var attempt int

		backoffState, cancel := policy.Start(base.Context)
		defer cancel()
		for backoff.Continue(backoffState) {
			if err = handler(message); err == nil {
				return nil
			}
			utils.LogError("err", err.Error(), "attempt", attempt, "Try failed")
			attempt++
		}
		return err
	}
}

type CreateReader func(topic string) Reader
type CreateWriter func(topic string) Writer

func runReader(wg *sync.WaitGroup, topic string, createReader CreateReader, msgHandler MessageHandler) {
	defer wg.Done()
	defer utils.LogPanics(true)
	reader := createReader(topic)
	defer reader.Close()
	reader.HandleMessages(msgHandler)
}

func SpawnReader(wg *sync.WaitGroup, topic string, createReader CreateReader, msgHandler MessageHandler) {
	wg.Add(1)
	go runReader(wg, topic, createReader, msgHandler)
}
