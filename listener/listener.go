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
	eventsTopic        string
	eventsConsumers    int
	enableTemplates    bool
	templatesTopic     string
	templatesConsumers int
	evalWriter         mqueue.Writer
	ptWriter           mqueue.Writer
	validReporters     map[string]int
	allowedReporters   map[string]bool
	excludedHostTypes  map[string]bool
	enableBypass       bool
	uploadEvalTimeout  time.Duration
)

func configure() {
	core.ConfigureApp()
	eventsTopic = utils.FailIfEmpty(utils.CoreCfg.EventsTopic, "EVENTS_TOPIC")
	eventsConsumers = utils.GetIntEnvOrDefault("CONSUMER_COUNT", 1)

	enableTemplates = utils.GetBoolEnvOrDefault("ENABLE_TEMPLATES_API", true)
	if enableTemplates {
		templatesTopic = utils.FailIfEmpty(utils.CoreCfg.TemplateTopic, "TEMPLATE_TOPIC")
		templatesConsumers = utils.GetIntEnvOrDefault("TEMPLATE_CONSUMERS", 1)
	}

	evalTopic := utils.FailIfEmpty(utils.CoreCfg.EvalTopic, "EVAL_TOPIC")
	ptTopic := utils.FailIfEmpty(utils.CoreCfg.PayloadTrackerTopic, "PAYLOAD_TRACKER_TOPIC")

	evalWriter = mqueue.NewKafkaWriterFromEnv(evalTopic)
	ptWriter = mqueue.NewKafkaWriterFromEnv(ptTopic)

	validReporters = loadValidReporters()
	allowedReporters = getEnvVarStringsSet("ALLOWED_REPORTERS")
	excludedHostTypes = getEnvVarStringsSet("EXCLUDED_HOST_TYPES")

	enableBypass = utils.GetBoolEnvOrDefault("ENABLE_BYPASS", false)

	uploadEvalTimeout = time.Duration(utils.GetIntEnvOrDefault("UPLOAD_EVAL_TIMEOUT_MS", 500)) * time.Millisecond
}

func getEnvVarStringsSet(envVarName string) map[string]bool {
	strValue := os.Getenv(envVarName)
	mapValue := make(map[string]bool)
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
	database.DB.Find(&reporters)
	reportersMap := make(map[string]int, len(reporters))
	for _, reporter := range reporters {
		reportersMap[reporter.Name] = reporter.ID
	}
	return reportersMap
}

func runReaders(wg *sync.WaitGroup, readerBuilder mqueue.CreateReader) {
	utils.LogInfo("listener starting")

	// Start a web server for handling metrics so that readiness probe works
	go RunMetrics()

	configure()

	// We create multiple consumers, and hope that the partition rebalancing
	// algorithm assigns each consumer a single partition
	for i := 0; i < eventsConsumers; i++ {
		mqueue.SpawnReader(wg, eventsTopic, readerBuilder, mqueue.MakeRetryingHandler(EventsMessageHandler))
		utils.LogDebug("spawned eventsTopic reader", i)
	}
	if enableTemplates {
		for i := 0; i < templatesConsumers; i++ {
			mqueue.SpawnReader(wg, templatesTopic, readerBuilder, mqueue.MakeRetryingHandler(TemplatesMessageHandler))
			utils.LogDebug("spawned templatesTopic reader", i)
		}
	}
	utils.LogInfo("connected to kafka topics")
}

func RunListener() {
	var wg sync.WaitGroup

	go utils.RunProfiler()

	runReaders(&wg, mqueue.NewKafkaReaderFromEnv)
	wg.Wait()
	utils.LogInfo("listener completed")
}
