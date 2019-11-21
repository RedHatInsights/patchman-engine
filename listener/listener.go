package listener

import (
	"app/base/utils"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

var (
	uploadReader *kafka.Reader
	eventsReader *kafka.Reader
)

func configure() {
	uploadTopic := utils.GetenvOrFail("UPLOAD_TOPIC")
	eventsTopic := utils.GetenvOrFail("EVENTS_TOPIC")

	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	kafkaGroup := utils.GetenvOrFail("KAFKA_GROUP")

	utils.Log("KafkaAddress", kafkaAddress).Info("Connecting to kafka")

	uploadConfig := kafka.ReaderConfig{
		Brokers:        []string{kafkaAddress},
		Topic:          uploadTopic,
		GroupID:        kafkaGroup,
		MinBytes:       1,
		MaxBytes:       10e6, // 1MB
	}

	uploadReader = kafka.NewReader(uploadConfig)

	eventsConfig := uploadConfig
	eventsConfig.Topic = eventsTopic

	eventsReader = kafka.NewReader(eventsConfig)

}

func shutdown() {
	err := uploadReader.Close()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to shutdown Kafka reader")
	}
	err = eventsReader.Close()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to shutdown Kafka reader")
	}

}

func baseListener(reader *kafka.Reader, handler func(message kafka.Message)) {
	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			utils.Log("err", err.Error()).Error("unable to read message from Kafka reader")
			panic(err)

		}
		handler(m)

	}
}

func logHandler(m kafka.Message) {
	utils.Log().Info("Received message [", m.Topic,"] ", string(m.Value))
}

func runMetrics() {
	// create web app
	app := gin.New()

	prometheus := ginprometheus.NewPrometheus("gin")
	prometheus.Use(app)
	err := app.Run(":8081")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}

func RunListener() {
	utils.Log().Info("listener starting")

	// Start a web server for handling metrics so that readiness probe works
	go runMetrics()

	configure()
	defer shutdown()

	go baseListener(uploadReader, logHandler)
	go baseListener(eventsReader, logHandler)

	// Just block. Any error will panic and kill the process.
	<- make(chan bool)

}
