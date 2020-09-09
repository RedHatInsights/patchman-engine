// Provides some basic methods for interacting with platform kafka message queue
package mqueue

import (
	"app/base"
	"app/base/utils"
	"context"
	"github.com/Shopify/sarama"
	"github.com/lestrrat-go/backoff"
	"io"
	"time"
)

type MessageHandler func(message Message) error

// By wrapping raw value we can add new methods & ensure methods of wrapped type are callable
type Reader interface {
	HandleMessages(handler MessageHandler)
	io.Closer
}

type readerImpl struct {
	sarama.ConsumerGroup
	topic string
}

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...Message) error
}

type writerImpl struct {
	sarama.SyncProducer
	topic string
}

func (w writerImpl) WriteMessages(_ context.Context, msgs ...Message) error {
	for _, m := range msgs {
		msg := sarama.ProducerMessage{
			Topic: w.topic,
			Key:   sarama.ByteEncoder(m.Key),
			Value: sarama.ByteEncoder(m.Value),
		}
		if _, _, err := w.SendMessage(&msg); err != nil {
			return err
		}
	}
	return nil
}

func ReaderFromEnv(topic string) Reader {
	addresses := []string{utils.GetenvOrFail("KAFKA_ADDRESS")}
	kafkaGroup := utils.GetenvOrFail("KAFKA_GROUP")
	minBytes := utils.GetIntEnvOrDefault("KAFKA_READER_MIN_BYTES", 1)
	maxBytes := utils.GetIntEnvOrDefault("KAFKA_READER_MAX_BYTES", 1e6)

	config := sarama.NewConfig()
	config.Version = sarama.V1_1_0_0

	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.MaxProcessingTime = 3 * time.Second
	config.Consumer.Fetch.Min = int32(minBytes)
	config.Consumer.Fetch.Max = int32(maxBytes)

	consumer, err := sarama.NewConsumerGroup(addresses, kafkaGroup, config)
	if err != nil {
		panic(err)
	}
	reader := &readerImpl{consumer, topic}
	return reader
}

func WriterFromEnv(topic string) Writer {
	addresses := []string{utils.GetenvOrFail("KAFKA_ADDRESS")}

	config := sarama.NewConfig()
	config.Version = sarama.V1_1_0_0
	config.Producer.Flush.Messages = 1
	config.Producer.Flush.Frequency = time.Millisecond

	// Must be set for sync producer
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	producer, err := sarama.NewSyncProducer(addresses, config)
	if err != nil {
		panic(err)
	}

	writer := &writerImpl{producer, topic}
	return writer
}

func MakeRetryingHandler(handler MessageHandler) MessageHandler {
	return func(message Message) error {
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

type messageConsumer struct {
	MessageHandler
}

func (consumer *messageConsumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *messageConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *messageConsumer) ConsumeClaim(session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim) error {
	for m := range claim.Messages() {
		msg := Message{
			Key:   m.Key,
			Value: m.Value,
		}
		if err := consumer.MessageHandler(msg); err != nil {
			utils.Log("err", err.Error()).Panic("Handler failed")
		}
		session.MarkMessage(m, "")
	}
	return nil
}

func (t *readerImpl) HandleMessages(handler MessageHandler) {
	for {
		consumer := messageConsumer{handler}
		if err := t.ConsumerGroup.Consume(base.Context, []string{t.topic}, &consumer); err != nil {
			utils.Log("err", err).Panic("Consumer error")
		}
	}
}

type CreateReader func(topic string) Reader

func RunReader(topic string, createReader CreateReader, msgHandler MessageHandler) {
	defer utils.LogPanics(true)
	reader := createReader(topic)
	defer reader.Close()
	reader.HandleMessages(msgHandler)
}
