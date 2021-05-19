// Provides some basic methods for interacting with platform kafka message queue
package mqueue

import (
	"app/base"
	"app/base/utils"
	"context"
	"github.com/lestrrat-go/backoff"
	"io"
	"strings"
	"sync"
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
		utils.Log("type", "kafka").Errorf(fmt, args...)
		if strings.Contains(fmt, "Group Load In Progress") {
			utils.Log().Panic("Kafka client stuck detected!!!")
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
			utils.Log("err", err.Error(), "attempt", attempt).Error("Try failed")
			attempt++
		}
		return err
	}
}

func (t *kafkaGoReaderImpl) HandleMessages(handler MessageHandler) {
	for {
		m, err := t.FetchMessage(base.Context)
		if err != nil {
			if err.Error() == errContextCanceled {
				break
			}
			utils.Log("err", err.Error()).Error("unable to read message from Kafka reader")
			panic(err)
		}
		// At this level, all errors are fatal
		kafkaMessage := KafkaMessage{Key: m.Key, Value: m.Value}
		if err = handler(kafkaMessage); err != nil {
			utils.Log("err", err.Error()).Panic("Handler failed")
		}
		err = t.CommitMessages(base.Context, m)
		if err != nil {
			if err.Error() == errContextCanceled {
				break
			}
			utils.Log("err", err.Error()).Error("unable to commit kafka message")
			panic(err)
		}
	}
}

type CreateReader func(topic string) Reader

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
