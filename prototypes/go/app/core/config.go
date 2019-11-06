package core

import (
	"gin-container/app/database"
	"gin-container/app/utils"
)

func ConfigureApp() {
	utils.ConfigureLogging()
	database.Configure()
}

func SetupTestEnvironment() {
	utils.SetenvOrFail("LOG_LEVEL", "debug")

	ConfigureApp()
	err := database.DelteAllHosts()
	if err != nil {
		panic(err)
	}
}
