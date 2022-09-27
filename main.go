package main

import (
	"app/base"
	"app/base/utils"
	"app/database_admin"
	"app/evaluator"
	"app/listener"
	"app/manager"
	"app/platform"
	"app/tasks/caches"
	"app/tasks/cleaning"
	"app/tasks/system_culling"
	"app/tasks/vmaas_sync"
	"app/turnpike"
	"log"
	"os"
)

// nolint: funlen
func main() {
	base.HandleSignals()

	defer utils.LogPanics(true)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "admin":
			turnpike.RunAdminAPI()
			return
		case "manager":
			manager.RunManager()
			return
		case "listener":
			listener.RunListener()
			return
		case "evaluator":
			evaluator.RunEvaluator()
			return
		case "migrate":
			database_admin.UpdateDB(os.Args[2])
			return
		case "platform":
			platform.RunPlatformMock()
			return
		case "print_clowder_params":
			utils.PrintClowderParams()
			return
		case "check_upgraded":
			database_admin.CheckUpgraded(os.Args[2])
			return
		case "job":
			runJob(os.Args[2])
			return
		}
	}
	log.Panic("You need to provide a command")
}

func runJob(name string) {
	switch name {
	case "vmaas_sync":
		vmaas_sync.RunVmaasSync()
	case "system_culling":
		system_culling.RunSystemCulling()
	case "advisory_cache_refresh":
		caches.RunAdvisoryRefresh()
	case "delete_unused":
		cleaning.RunDeleteUnusedData()
	}
}
