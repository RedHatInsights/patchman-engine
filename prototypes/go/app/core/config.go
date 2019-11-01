package core

import (
	"gin-container/app/database"
	"gin-container/app/utils"
)

// configure SDN using given env values (address, service_id, password, and httpClient)
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
