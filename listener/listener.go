package listener

import (
	"app/base/mqueue"
	"app/base/utils"
	"sync"
)

var (
	uploadTopic   string
	eventsTopic   string
	consumerCount int
	evalWriter    mqueue.Writer
)

func configure() {
	uploadTopic = utils.GetenvOrFail("UPLOAD_TOPIC")
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

	// We create multiple consumers, and hope that the partition rebalancing
	// algorithm assigns each consumer a single partition
	for i := 0; i < consumerCount; i++ {
		mqueue.RunReader(wg, uploadTopic, readerBuilder, mqueue.MakeRetryingHandler(uploadMsgHandler))
		mqueue.RunReader(wg, eventsTopic, readerBuilder, mqueue.MakeRetryingHandler(DeleteMessageHandler))
	}
}

func RunListener() {
	var wg sync.WaitGroup
	runReaders(&wg, mqueue.ReaderFromEnv)
	wg.Wait()
	utils.Log().Info("listener completed")
}
