package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
)

var (
	eventsTopic    string
	evalWriter     mqueue.Writer
	validReporters map[string]int
)

func configure() {
	core.ConfigureApp()
	eventsTopic = utils.GetenvOrFail("EVENTS_TOPIC")
	evalTopic := utils.GetenvOrFail("EVAL_TOPIC")

	evalWriter = mqueue.WriterFromEnv(evalTopic)

	validReporters = loadValidReporters()
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

func runReaders(readerBuilder mqueue.CreateReader) {
	utils.Log().Info("listener starting")

	// Start a web server for handling metrics so that readiness probe works
	go RunMetrics()

	configure()

	mqueue.RunReader(eventsTopic, readerBuilder, mqueue.MakeRetryingHandler(EventsMessageHandler))
}

func RunListener() {
	runReaders(mqueue.ReaderFromEnv)
	utils.Log().Info("listener completed")
}
