// Provides some basic methods for interacting with platform kafka message queue
package mqueue

import (
	"app/base/utils"
	"context"
	"github.com/segmentio/kafka-go"
	"time"
)

// By wrapping raw value we can add new methods & ensure methods of wrapped type are callable
type Reader struct {
	kafka.Reader
}
type Writer struct {
	kafka.Writer
}

func NewReader(topic string) *Reader {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	kafkaGroup := utils.GetenvOrFail("KAFKA_GROUP")

	config := kafka.ReaderConfig{
		Brokers:  []string{kafkaAddress},
		Topic:    topic,
		GroupID:  kafkaGroup,
		MinBytes: 1,
		MaxBytes: 10e6, // 1MB
		MaxWait:  time.Second * 30,
		ErrorLogger: kafka.LoggerFunc(func(fmt string, args ...interface{}) {
			utils.Log("type", "kafka").Errorf(fmt, args)
		}),
	}

	return &Reader{*kafka.NewReader(config)}
}

func NewWriter(topic string) *Writer {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")

	config := kafka.WriterConfig{
		Brokers: []string{kafkaAddress},
		Topic:   topic,
		ErrorLogger: kafka.LoggerFunc(func(fmt string, args ...interface{}) {
			utils.Log("type", "kafka").Errorf(fmt, args)
		}),
	}

	return &Writer{*kafka.NewWriter(config)}
}

func (t *Reader) Shutdown() {
	err := t.Close()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to shutdown Kafka reader")
	}
}

type KafkaHandler func(message kafka.Message)

func (t *Reader) HandleMessages(handler KafkaHandler) {
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
			utils.Log("err", err.Error()).Error("unable to read message from Kafka reader")
			panic(err)
		}
	}
}
