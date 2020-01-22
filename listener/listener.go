package listener

import (
	"app/base"
	"app/base/mqueue"
	"app/base/utils"
	"app/evaluator"
	"github.com/RedHatInsights/patchman-clients/inventory"
)

var (
	uploadReader    *mqueue.Reader
	eventsReader    *mqueue.Reader
	inventoryClient *inventory.APIClient
)

func configure() {
	uploadTopic := utils.GetenvOrFail("UPLOAD_TOPIC")
	eventsTopic := utils.GetenvOrFail("EVENTS_TOPIC")

	uploadReader = mqueue.ReaderFromEnv(uploadTopic)
	eventsReader = mqueue.ReaderFromEnv(eventsTopic)

	traceAPI := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	inventoryAddr := utils.GetenvOrFail("INVENTORY_ADDRESS")

	inventoryConfig := inventory.NewConfiguration()
	inventoryConfig.Debug = traceAPI
	inventoryConfig.BasePath = inventoryAddr + base.InventoryAPIPrefix
	inventoryClient = inventory.NewAPIClient(inventoryConfig)

	evaluator.Configure() // TODO - move to evaluator component
}

func RunListener() {
	utils.Log().Info("listener starting")

	// Start a web server for handling metrics so that readiness probe works
	go RunMetrics()

	configure()

	defer uploadReader.Shutdown()
	// Only respond to creation and update msgs on upload topic
	go uploadReader.HandleEvents(uploadHandler)

	defer eventsReader.Shutdown()
	// Only respond to deletion on events topic
	eventsReader.HandleEvents(deleteHandler)
}
