package core

import (
	"app/base/database"
	"app/base/metrics"
	"app/base/utils"
	"testing"
)

var (
	DefaultLimit  = 20
	DefaultOffset = 0
	testSetupRan  = false
	dbWait        = utils.PodConfig.GetString("wait_for_db", "empty")
)

func ConfigureApp() {
	utils.ConfigureLogging()
	database.Configure()
	metrics.Configure()
	database.DBWait(dbWait)
}

func SetupTestEnvironment() {
	utils.SetDefaultEnvOrFail("LOG_LEVEL", "debug")
	ConfigureApp()
}

func SetupTest(t *testing.T) {
	if !testSetupRan {
		utils.SkipWithoutDB(t)
		SetupTestEnvironment()
		testSetupRan = true
	}
}
