package listener

import (
	"app/base"
	"app/base/utils"
	"app/manager/middlewares"
	"context"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
)

var (
	uploadReader    *kafka.Reader
	eventsReader    *kafka.Reader
	inventoryClient *inventory.APIClient
	vmaasClient     *vmaas.APIClient
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

	vmaasConfig := vmaas.NewConfiguration()
	vmaasConfig.BasePath = utils.GetenvOrFail("VMAAS_ADDRESS") + base.VMAAS_API_PREFIX
	vmaasConfig.Debug = traceApi
	vmaasClient = vmaas.NewAPIClient(vmaasConfig)
}

func shutdown(reader *kafka.Reader) {
	err := reader.Close()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to shutdown Kafka reader")
	}
}

func baseListener(reader *kafka.Reader, handler func(message kafka.Message)) {
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

func logHandler(m kafka.Message) {
	utils.Log("topic", m.Topic, "value", string(m.Value)).Info("Received message ")
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

	go baseListener(uploadReader, uploadHandler)
	baseListener(eventsReader, logHandler)
}
