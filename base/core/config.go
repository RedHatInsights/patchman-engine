package core

import (
	"app/base/database"
	"app/base/metrics"
	"app/base/telemetry"
	"app/base/utils"
	"testing"
)

var (
	DefaultLimit  = 20
	DefaultOffset = 0
	testSetupRan  = false
	dbWait        = utils.PodConfig.GetString("wait_for_db", "empty")
)

func configureBaseApp() {
	utils.ConfigureLogging()
	if err := telemetry.Init(); err != nil {
		panic(err)
	}
	metrics.Configure()
	database.DBWait(dbWait)
}

func ConfigureApp() {
	database.Configure()
	configureBaseApp()
}

func ConfigureAdminApp() {
	database.ConfigureAdmin()
	configureBaseApp()
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
