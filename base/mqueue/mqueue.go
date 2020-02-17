// Provides some basic methods for interacting with platform kafka message queue
package mqueue

import (
	"app/base/utils"
	"context"
	"github.com/segmentio/kafka-go"
	"io"
)

// By wrapping raw value we can add new methods & ensure methods of wrapped type are callable
type Reader interface {
	HandleEvents(handler EventHandler)
	io.Closer
}

type readerImpl struct {
	kafka.Reader
}

type Writer interface {
	WriteEvents(ctx context.Context, events ...PlatformEvent) error
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
		ErrorLogger: kafka.LoggerFunc(func(fmt string, args ...interface{}) {
			utils.Log("type", "kafka").Errorf(fmt, args)
		}),
	}
	writer := &writerImpl{kafka.NewWriter(config)}
	return writer
}

type KafkaHandler func(message kafka.Message)

func (t *readerImpl) HandleMessages(handler KafkaHandler) {
	ctx := context.Background()

	for {
		m, err := t.FetchMessage(ctx)
		if err != nil {
			utils.Log("err", err.Error()).Error("unable to read message from Kafka reader")
			panic(err)
		}
		// Process synchronously, for now lower performance, but we ensure correct committing of messages
		handler(m)
		err = t.CommitMessages(ctx, m)
		if err != nil {
			utils.Log("err", err.Error()).Error("unable to commit kafka message")
			panic(err)
		}
	}
}

type CreateReader func(topic string) Reader

func RunReader(topic string, createReader CreateReader, eventHandler EventHandler) {
	reader := createReader(topic)
	defer reader.Close()
	reader.HandleEvents(eventHandler)
}
