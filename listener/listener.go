package listener

import (
	"app/base"
	"app/base/utils"
	"app/evaluator"
	"app/manager/middlewares"
	"context"
	"encoding/json"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
)

var (
	uploadReader    *kafka.Reader
	eventsReader    *kafka.Reader
	inventoryClient *inventory.APIClient
)

func configure() {
	uploadTopic := utils.GetenvOrFail("UPLOAD_TOPIC")
	eventsTopic := utils.GetenvOrFail("EVENTS_TOPIC")

	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	kafkaGroup := utils.GetenvOrFail("KAFKA_GROUP")

	utils.Log("KafkaAddress", kafkaAddress).Info("Connecting to kafka")

	uploadConfig := kafka.ReaderConfig{
		Brokers:  []string{kafkaAddress},
		Topic:    uploadTopic,
		GroupID:  kafkaGroup,
		MinBytes: 1,
		MaxBytes: 10e6, // 1MB
	}

	uploadReader = kafka.NewReader(uploadConfig)

	eventsConfig := uploadConfig
	eventsConfig.Topic = eventsTopic

	eventsReader = kafka.NewReader(eventsConfig)

	traceApi := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	inventoryAddr := utils.GetenvOrFail("INVENTORY_ADDRESS")

	inventoryConfig := inventory.NewConfiguration()
	inventoryConfig.Debug = traceApi
	inventoryConfig.BasePath = inventoryAddr + base.INVENTORY_API_PREFIX
	inventoryClient = inventory.NewAPIClient(inventoryConfig)

	evaluator.Configure() // TODO - move to evaluator component
}

func shutdown(reader *kafka.Reader) {
	err := reader.Close()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to shutdown Kafka reader")
	}
}

type KafkaHandler func(message kafka.Message)
type EventHandler func(event PlatformEvent)

func baseListener(reader *kafka.Reader, handler KafkaHandler) {
	defer shutdown(reader)

	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			utils.Log("err", err.Error()).Error("unable to read message from Kafka reader")
			panic(err)
		}
		// Spawn handler, not blocking the receiving goroutine
		go handler(m)
	}
}

// Performs parsing of kafka message, and then dispatches this message into provided functions
func makeKafkaHandler(eventHandler EventHandler) KafkaHandler {
	return func(m kafka.Message) {
		var event PlatformEvent
		err := json.Unmarshal(m.Value, &event)
		if err != nil {
			utils.Log("err", err.Error()).Error("Could not deserialize platform event")
			return
		}
		eventHandler(event)
	}
}

func runMetrics() {
	// create web app
	app := gin.New()
	middlewares.Prometheus().Use(app)

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

	// Only respond to creation and update msgs on upload topic
	go baseListener(uploadReader, makeKafkaHandler(uploadHandler))

	// Only respond to deletion on events topic
	baseListener(eventsReader, makeKafkaHandler(deleteHandler))
}
