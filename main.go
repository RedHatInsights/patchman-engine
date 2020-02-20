package main

import (
	"app/base"
	"app/base/core"
	"app/base/utils"
	"app/database_admin"
	"app/evaluator"
	"app/listener"
	"app/manager"
	"app/vmaas_sync"
	"log"
	"os"
)

func main() {
	base.HandleSignals()
	core.ConfigureApp()

	defer utils.LogPanicsAndExit()
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
		}
	}
	log.Fatal("You need to provide a command")
}
