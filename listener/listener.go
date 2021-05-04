package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	eventsTopic       string
	consumerCount     int
	evalWriter        mqueue.Writer
	validReporters    map[string]int
	excludedReporters map[string]bool
	enableBypass      bool
	uploadEvalTimeout time.Duration
)

func configure() {
	core.ConfigureApp()
	eventsTopic = utils.GetenvOrFail("EVENTS_TOPIC")

	consumerCount = utils.GetIntEnvOrFail("CONSUMER_COUNT")

	evalTopic := utils.GetenvOrFail("EVAL_TOPIC")

	evalWriter = mqueue.WriterFromEnv(evalTopic)

	validReporters = loadValidReporters()
	excludedReporters = loadExcludedReporters()

	enableBypass = utils.GetBoolEnvOrDefault("ENABLE_BYPASS", false)

	uploadEvalTimeout = time.Duration(utils.GetIntEnvOrDefault("UPLOAD_EVAL_TIMEOUT_MS", 500)) * time.Millisecond
}

func loadExcludedReporters() map[string]bool {
	excludeReportersStr := os.Getenv("EXCLUDED_REPORTERS")
	arr := strings.Split(excludeReportersStr, ",")
	excludedReporters := map[string]bool{}
	for _, m := range arr {
		excludedReporters[m] = true
	}
	return excludedReporters
}

func loadValidReporters() map[string]int {
	var reporters []models.Reporter
	database.Db.Find(&reporters)
	reportersMap := map[string]int{}
	for _, reporter := range reporters {
		reportersMap[reporter.Name] = reporter.ID
	}
	return reportersMap
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
