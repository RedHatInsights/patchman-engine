package listener

import (
	"context"
	"encoding/json"
	"gin-container/app/database"
	"gin-container/app/structures"
	"gin-container/app/utils"
	"github.com/segmentio/kafka-go"
	"time"
)

var (
	kafkaReader       *kafka.Reader
	storage           *Storage
	benchmarkMessages int
	useBatchWrite     bool
)

func configure() {
	topic := utils.GetenvOrFail("LISTENER_KAFKA_TOPIC")
	kafkaAddress := utils.GetenvOrFail("LISTENER_KAFKA_ADDRESS")

	kafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaAddress},
		Topic:	   topic,
		Partition: 0,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
	})

	bufferSize := utils.GetIntEnvOrFail("LISTENER_BUFFER_SIZE")
	useBatchWrite = utils.GetenvOrFail("LISTENER_BATCH_WRITE") == "on"
	storage = InitStorage(bufferSize, useBatchWrite)
	benchmarkMessages = utils.GetIntEnvOrFail("BENCHMARK_MESSAGES")
}

func shutdown() {
	err := kafkaReader.Close()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to shutdown Kafka reader")
	}
}

func RunListener() {
	utils.Log().Info("listener starting")
	configure()
	defer shutdown()

	err := database.DelteAllHosts() // start with clean database, remove existing items
	if err != nil {
		panic(err)
	}

	var msg Message // struct to parse Kafka message into
	var host structures.HostDAO // struct to store parsed msg from Kafka

	// Benchmark
	benchmark := InitBenchmark(benchmarkMessages, storage)

	for {
		m, err := kafkaReader.ReadMessage(context.Background())
		if err != nil {
			if err.Error() == "context deadline exceeded" {
				utils.Log().Info("waiting for messages")
				time.Sleep(time.Second)
				continue
			}

			utils.Log("err", err.Error()).Error("unable to read message from Kafka reader")
			continue
		}

		err = json.Unmarshal(m.Value, &msg)
		if err != nil {
			utils.Log("err", err.Error()).Error("unable to parse message from Kafka reader")
			continue
		}

		msg.FilterPackages()

		host.ID = msg.ID
		host.Request = string(msg.ToJSON())
		host.Checksum = msg.JSONChecksum()

		err = storage.Add(&host)
		if err != nil {
			utils.Log("err", err.Error()).Error("unable to add item to storage")
		}

		benchmark.Increment()
	}
}
