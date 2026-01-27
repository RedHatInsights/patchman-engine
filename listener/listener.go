package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	eventsTopic          string
	eventsConsumers      int
	enableTemplates      bool
	templatesTopic       string
	templatesConsumers   int
	evalWriter           mqueue.Writer
	createdSystemsWriter mqueue.Writer
	ptWriter             mqueue.Writer
	validReporters       map[string]int
	allowedReporters     map[string]bool
	excludedHostTypes    map[string]bool
	enableBypass         bool
	uploadEvalTimeout    time.Duration
	deletionThreshold    time.Duration
	useTraceLevel        bool
)

const (
	puptooReporter = "puptoo"
	rhsmReporter   = "rhsm-system-profile-bridge"
)

func configure() {
	core.ConfigureApp()
	eventsTopic = utils.FailIfEmpty(utils.CoreCfg.EventsTopic, "EVENTS_TOPIC")
	evalTopic := utils.FailIfEmpty(utils.CoreCfg.EvalTopic, "EVAL_TOPIC")
	createdTopic := utils.FailIfEmpty(utils.CoreCfg.CreatedSystemsTopic, "CREATED_SYSTEMS_TOPIC")
	ptTopic := utils.FailIfEmpty(utils.CoreCfg.PayloadTrackerTopic, "PAYLOAD_TRACKER_TOPIC")
	evalWriter = mqueue.NewKafkaWriterFromEnv(evalTopic)
	ptWriter = mqueue.NewKafkaWriterFromEnv(ptTopic)
	createdSystemsWriter = mqueue.NewKafkaWriterFromEnv(createdTopic)

	configureListener()
}

func configureListener() {
	// Number of kafka readers for upload topic
	eventsConsumers = utils.PodConfig.GetInt("consumer_count", 1)
	// Toggle template message processing
	enableTemplates = utils.PodConfig.GetBool("templates_api", true)
	if enableTemplates {
		// Name of kafka template topic
		templatesTopic = utils.FailIfEmpty(utils.CoreCfg.TemplateTopic, "TEMPLATE_TOPIC")
		// Number of kafka readers for template topic
		templatesConsumers = utils.PodConfig.GetInt("template_consumers", 1)
	}
	// Comma-separated list of reporters to include into processing
	reporters := strings.Join([]string{puptooReporter, rhsmReporter}, ",")
	allowedReporters = utils.PodConfig.GetStringSet("allowed_reporters", reporters)
	// Comma-separated list of host types to exclude from processing
	excludedHostTypes = utils.PodConfig.GetStringSet("excluded_host_types", "edge")
	// Toggle bypass (fake) messages processing
	enableBypass = utils.PodConfig.GetBool("bypass", false)
	// How long to collect upload messages before grouping them and sending to evaluator
	uploadEvalTimeout = time.Duration(utils.PodConfig.GetInt("upload_eval_timeout_ms", 500)) * time.Millisecond
	// Ignore a system if there was a delete message in the last X hours
	deletionThreshold = time.Hour * time.Duration(utils.PodConfig.GetInt("system_delete_hrs", 4))
	useTraceLevel = log.IsLevelEnabled(log.TraceLevel)

	validReporters = loadValidReporters()

	updatedEventsBuffer.initFlushTimer(&evalWriter)
	createdEventsBuffer.initFlushTimer(&createdSystemsWriter)
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
