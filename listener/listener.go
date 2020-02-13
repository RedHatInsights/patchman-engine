package listener

import (
	"app/base"
	"app/base/mqueue"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/inventory"
)

var (
	uploadTopic     string
	eventsTopic     string
	consumerCount   int
	evalWriter      *mqueue.Writer
	inventoryClient *inventory.APIClient
)

func configure() {
	uploadTopic = utils.GetenvOrFail("UPLOAD_TOPIC")
	eventsTopic = utils.GetenvOrFail("EVENTS_TOPIC")

	consumerCount = utils.GetIntEnvOrFail("CONSUMER_COUNT")

	evalTopic := utils.GetenvOrFail("EVAL_TOPIC")

	evalWriter = mqueue.WriterFromEnv(evalTopic)

	traceAPI := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	inventoryAddr := utils.GetenvOrFail("INVENTORY_ADDRESS")

	inventoryConfig := inventory.NewConfiguration()
	inventoryConfig.Debug = traceAPI
	inventoryConfig.BasePath = inventoryAddr + base.InventoryAPIPrefix
	inventoryClient = inventory.NewAPIClient(inventoryConfig)
}

type ReaderBuilder func(topic string) *mqueue.Reader

func runReader(topic string, readerBuilder ReaderBuilder, handler mqueue.EventHandler) {
	reader := readerBuilder(topic)
	defer (*reader).Shutdown()
	(*reader).HandleEvents(handler)
}

func runReaders(readerBuilder ReaderBuilder) {
	utils.Log().Info("listener starting")

	// Start a web server for handling metrics so that readiness probe works
	go RunMetrics()

	configure()

	// We create multiple consumers, and hope that the partition rebalancing
	// algorithm assigns each consumer a single partition
	for i := 0; i < consumerCount; i++ {
		go runReader(uploadTopic, readerBuilder, uploadHandler)
		go runReader(eventsTopic, readerBuilder, deleteHandler)
	}
}

func RunListener() {
	runReaders(mqueue.ReaderFromEnv)
	<-make(chan bool)
}
