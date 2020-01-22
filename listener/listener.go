package listener

import (
	"app/base"
	"app/base/mqueue"
	"app/base/utils"
	"app/evaluator"
	"app/manager/middlewares"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/gin-gonic/gin"
)

var (
	uploadReader    *mqueue.Reader
	eventsReader    *mqueue.Reader
	inventoryClient *inventory.APIClient
)

func configure() {
	uploadTopic := utils.GetenvOrFail("UPLOAD_TOPIC")
	eventsTopic := utils.GetenvOrFail("EVENTS_TOPIC")

	uploadReader = mqueue.NewReader(uploadTopic)
	eventsReader = mqueue.NewReader(eventsTopic)

	traceAPI := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	inventoryAddr := utils.GetenvOrFail("INVENTORY_ADDRESS")

	inventoryConfig := inventory.NewConfiguration()
	inventoryConfig.Debug = traceAPI
	inventoryConfig.BasePath = inventoryAddr + base.InventoryAPIPrefix
	inventoryClient = inventory.NewAPIClient(inventoryConfig)

	evaluator.Configure() // TODO - move to evaluator component
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

	defer uploadReader.Shutdown()
	// Only respond to creation and update msgs on upload topic
	go uploadReader.HandleEvents(uploadHandler)

	defer eventsReader.Shutdown()
	// Only respond to deletion on events topic
	go eventsReader.HandleEvents(deleteHandler)
	<-make(chan bool)
}
