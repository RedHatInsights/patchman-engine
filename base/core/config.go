package core

import (
	"app/base/database"
	"app/base/metrics"
	"app/base/utils"
)

var (
	DefaultLimit = 20
)

func ConfigureApp() {
	utils.ConfigureLogging()
	database.Configure()
	metrics.Configure()
	database.DBWait(utils.Getenv("WAIT_FOR_DB", "UNSET"))
}

func SetupTestEnvironment() {
	utils.SetDefaultEnvOrFail("LOG_LEVEL", "debug")
	ConfigureApp()
}
