package listener

import (
	"app/base/core"
	"app/base/mqueue"
	"app/base/utils"
	"sync"
)

var (
	eventsTopic   string
	consumerCount int
	evalWriter    mqueue.Writer
)

func configure() {
	core.ConfigureApp()
	eventsTopic = utils.GetenvOrFail("EVENTS_TOPIC")

	consumerCount = utils.GetIntEnvOrFail("CONSUMER_COUNT")

	evalTopic := utils.GetenvOrFail("EVAL_TOPIC")

	evalWriter = mqueue.WriterFromEnv(evalTopic)
}

func runReaders(wg *sync.WaitGroup, readerBuilder mqueue.CreateReader) {
	utils.Log().Info("listener starting")

	// Start a web server for handling metrics so that readiness probe works
	go RunMetrics()

	configure()
	mqueue.SpawnTimedReaderGroup(wg, consumerCount, eventsTopic, readerBuilder, mqueue.MakeRetryingHandler(EventsMessageHandler))

}

func RunListener() {
	var wg sync.WaitGroup
	runReaders(&wg, mqueue.ReaderFromEnv)
	wg.Wait()
	utils.Log().Info("listener completed")
}
