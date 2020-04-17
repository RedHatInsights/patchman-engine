// Provides some basic methods for interacting with platform kafka message queue
package mqueue

import (
	"app/base"
	"app/base/utils"
	"context"
	"github.com/lestrrat-go/backoff"
	"github.com/segmentio/kafka-go"
	"io"
	"sync"
	"time"
)

const errContextCanceled = "context canceled"

// By wrapping raw value we can add new methods & ensure methods of wrapped type are callable
type Reader interface {
	HandleMessages(handler MessageHandler)
	io.Closer
}

type readerImpl struct {
	kafka.Reader
}

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type writerImpl struct {
	*kafka.Writer
}

func ReaderFromEnv(topic string) Reader {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	kafkaGroup := utils.GetenvOrFail("KAFKA_GROUP")

	config := kafka.ReaderConfig{
		Brokers:  []string{kafkaAddress},
		Topic:    topic,
		GroupID:  kafkaGroup,
		MinBytes: 1,
		MaxBytes: 1e6, // 1MB
		ErrorLogger: kafka.LoggerFunc(func(fmt string, args ...interface{}) {
			utils.Log("type", "kafka").Errorf(fmt, args)
		}),
	}

	reader := &readerImpl{*kafka.NewReader(config)}
	return reader
}

func WriterFromEnv(topic string) Writer {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")

	config := kafka.WriterConfig{
		Brokers: []string{kafkaAddress},
		Topic:   topic,
		// By default the writer will wait for a second (or until the buffer is filled by different goroutines)
		// before sending the batch of messages. Disable this, and use it in 'non-batched' mode
		// meaning single messages are sent immediately for now. We'll maybe change this later if the
		// sending overhead is a bottleneck
		BatchTimeout: time.Nanosecond,
		ErrorLogger: kafka.LoggerFunc(func(fmt string, args ...interface{}) {
			utils.Log("type", "kafka").Errorf(fmt, args)
		}),
	}
	writer := &writerImpl{kafka.NewWriter(config)}
	return writer
}

type MessageHandler func(message kafka.Message) error

func MakeRetryingHandler(handler MessageHandler) MessageHandler {
	return func(message kafka.Message) error {
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

func (t *readerImpl) HandleMessages(handler MessageHandler) {
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
		if err = handler(m); err != nil {
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
	defer utils.LogPanicsAndExit()
	reader := createReader(topic)
	defer reader.Close()
	reader.HandleMessages(msgHandler)
}

func SpawnReader(wg *sync.WaitGroup, topic string, createReader CreateReader, msgHandler MessageHandler) {
	wg.Add(1)
	go runReader(wg, topic, createReader, msgHandler)
}
