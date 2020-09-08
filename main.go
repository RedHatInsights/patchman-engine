package main

import (
	"app/base"
	"app/base/utils"
	"app/database_admin"
	"app/evaluator"
	"app/listener"
	"app/manager"
	"app/platform"
	"app/vmaas_sync"
	"log"
	"os"
)

func main() {
	base.HandleSignals()

	defer utils.LogPanics(true)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "manager":
			manager.RunManager()
			return
		case "listener":
			listener.RunListener()
			return
		case "evaluator":
			evaluator.RunEvaluator()
			return
		case "vmaas_sync":
			vmaas_sync.RunVmaasSync()
			return
		case "migrate":
			database_admin.MigrateUp(os.Args[2], os.Args[3])
			return
		case "platform":
			platform.RunPlatformMock()
			return
		}
	}
	log.Panic("You need to provide a command")
}
