package mqueue

import (
	"app/base/utils"
	"context"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type confluenticReaderImpl struct {
	Topic    string
	Consumer *kafka.Consumer
}

func (t *confluenticReaderImpl) HandleMessages(handler MessageHandler) {
	for {
		m, err := t.Consumer.ReadMessage(-1)
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
		_, err = t.Consumer.CommitMessage(m)
		if err != nil {
			if err.Error() == errContextCanceled {
				break
			}
			utils.Log("err", err.Error()).Error("unable to commit kafka message")
			panic(err)
		}
	}
}

func (t *confluenticReaderImpl) Close() error {
	return t.Consumer.Close()
}

type confluenticWriterImpl struct {
	Topic    string
	Producer *kafka.Producer
}

func (t *confluenticWriterImpl) WriteMessages(ctx context.Context, msgs ...KafkaMessage) error {
	for _, m := range msgs {
		err := t.Producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &t.Topic, Partition: kafka.PartitionAny},
			Value:          m.Value,
		}, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func newConfluentReaderFromEnv(topic string) Reader {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	kafkaGroup := utils.GetenvOrFail("KAFKA_GROUP")
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaAddress,
		"group.id":          kafkaGroup,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		panic(err)
	}
	err = consumer.Subscribe(topic, nil)

	if err != nil {
		panic(err)
	}

	reader := &confluenticReaderImpl{topic, consumer}
	return reader
}

func newConfluentWriterFromEnv(topic string) Writer {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": kafkaAddress})
	if err != nil {
		panic(err)
	}

	writer := confluenticWriterImpl{topic, producer}
	return &writer
}
