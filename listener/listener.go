package listener

import (
	"app/base/mqueue"
	"app/base/utils"
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

func runReaders(readerBuilder mqueue.CreateReader) {
	utils.Log().Info("listener starting")

	// Start a web server for handling metrics so that readiness probe works
	go RunMetrics()

	configure()

	// We create multiple consumers, and hope that the partition rebalancing
	// algorithm assigns each consumer a single partition
	for i := 0; i < consumerCount; i++ {
		go mqueue.RunReader(uploadTopic, readerBuilder, uploadMsgHandler)
		go mqueue.RunReader(eventsTopic, readerBuilder, DeleteMessageHandler)
	}
}

func RunListener() {
	runReaders(mqueue.ReaderFromEnv)
	<-make(chan bool)
}
