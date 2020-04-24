package core

import (
	"app/base/database"
	"app/base/metrics"
	"app/base/utils"
)

var (
	DefaultLimit = 25
)

func ConfigureApp() {
	utils.ConfigureLogging()
	database.Configure()
	metrics.Configure()
}

func SetupTestEnvironment() {
	utils.SetenvOrFail("LOG_LEVEL", "debug")
	ConfigureApp()
}
