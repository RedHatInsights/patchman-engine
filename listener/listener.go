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
	ptWriter          mqueue.Writer
	validReporters    map[string]int
	excludedReporters map[string]bool
	excludedHostTypes map[string]bool
	enableBypass      bool
	uploadEvalTimeout time.Duration
)

func configure() {
	core.ConfigureApp()
	eventsTopic = utils.FailIfEmpty(utils.Cfg.EventsTopic, "EVENTS_TOPIC")

	consumerCount = utils.GetIntEnvOrDefault("CONSUMER_COUNT", 1)

	evalTopic := utils.FailIfEmpty(utils.Cfg.EvalTopic, "EVAL_TOPIC")
	ptTopic := utils.FailIfEmpty(utils.Cfg.PayloadTrackerTopic, "PAYLOAD_TRACKER_TOPIC")

	evalWriter = mqueue.NewKafkaWriterFromEnv(evalTopic)
	ptWriter = mqueue.NewKafkaWriterFromEnv(ptTopic)

	validReporters = loadValidReporters()
	excludedReporters = getEnvVarStringsSet("EXCLUDED_REPORTERS")
	excludedHostTypes = getEnvVarStringsSet("EXCLUDED_HOST_TYPES")

	enableBypass = utils.GetBoolEnvOrDefault("ENABLE_BYPASS", false)

	uploadEvalTimeout = time.Duration(utils.GetIntEnvOrDefault("UPLOAD_EVAL_TIMEOUT_MS", 500)) * time.Millisecond
}

func getEnvVarStringsSet(envVarName string) map[string]bool {
	strValue := os.Getenv(envVarName)
	mapValue := map[string]bool{}
	if strValue == "" {
		return mapValue
	}
	arr := strings.Split(strValue, ",")

	for _, m := range arr {
		mapValue[m] = true
	}
	return mapValue
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
	runReaders(&wg, mqueue.NewKafkaReaderFromEnv)
	wg.Wait()
	utils.Log().Info("listener completed")
}
