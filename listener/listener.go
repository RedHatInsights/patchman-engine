package listener

import (
	"app/base/core"
	"app/base/mqueue"
	"app/base/utils"
	"strings"
	"sync"
)

var (
	eventsTopic    string
	consumerCount  int
	evalWriter     mqueue.Writer
	validReporters map[string]bool
)

func configure() {
	core.ConfigureApp()
	eventsTopic = utils.GetenvOrFail("EVENTS_TOPIC")

	consumerCount = utils.GetIntEnvOrFail("CONSUMER_COUNT")

	evalTopic := utils.GetenvOrFail("EVAL_TOPIC")

	evalWriter = mqueue.WriterFromEnv(evalTopic)

	parseValidReporters()
}

func parseValidReporters() {
	arr := strings.Split(utils.Getenv("VALID_REPORTERS", "puptoo;rhsm-conduit;yupana"), ";")
	validReporters = map[string]bool{}
	for _, reporter := range arr {
		validReporters[reporter] = true
	}
}

func runReaders(wg *sync.WaitGroup, readerBuilder mqueue.CreateReader) {
	utils.Log().Info("listener starting")

	// Start a web server for handling metrics so that readiness probe works
	go RunMetrics()

	configure()

	// We create multiple consumers, and hope that the partition rebalancing
	// algorithm assigns each consumer a single partition
	for i := 0; i < consumerCount; i++ {
		mqueue.SpawnReader(wg, eventsTopic, readerBuilder, mqueue.MakeRetryingHandler(EventsMessageHandler))
	}
}

func RunListener() {
	var wg sync.WaitGroup
	runReaders(&wg, mqueue.ReaderFromEnv)
	wg.Wait()
	utils.Log().Info("listener completed")
}
