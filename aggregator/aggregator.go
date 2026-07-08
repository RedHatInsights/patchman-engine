package aggregator

import (
	"app/base"
	"app/base/core"
	"app/base/mqueue"
	"app/base/utils"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	advisoryUpdateTopic string
	enableNotifications bool
	consumerCount       int
)

func readPodConfig() {
	consumerCount = utils.PodConfig.GetInt("consumer_count", 1)
	enableNotifications = utils.PodConfig.GetBool("instant_notifications", false)
}

func configure() {
	advisoryUpdateTopic = utils.FailIfEmpty(utils.CoreCfg.AdvisoryUpdateTopic, "ADVISORY_UPDATE_TOPIC")
	configureNotifications()
}

func runServer() {
	app := gin.New()
	core.InitProbes(app)
	go base.TryExposeOnMetricsPort(app)

	err := utils.RunServer(base.Context, app, utils.CoreCfg.PublicPort)
	if err != nil {
		utils.LogError("err", err.Error())
		panic(err)
	}
}

func subscribeToAdvisoryUpdates(wg *sync.WaitGroup, readerBuilder mqueue.CreateReader) {
	handler := mqueue.MakeRetryingHandler(advisoryUpdateHandler)
	for i := 0; i < consumerCount; i++ {
		mqueue.SpawnReader(base.Context, wg, advisoryUpdateTopic, readerBuilder, handler)
		utils.LogDebug("spawned advisory update reader", i)
	}
	utils.LogInfo("connected to kafka topics")
}

func RunAggregator() {
	var wg sync.WaitGroup

	utils.LogInfo("aggregator starting")
	core.ConfigureApp()
	readPodConfig()
	configure()

	go runServer()
	go utils.RunProfiler()
	subscribeToAdvisoryUpdates(&wg, mqueue.NewKafkaReaderFromEnv)

	wg.Wait()
	utils.LogInfo("aggregator completed")
}
