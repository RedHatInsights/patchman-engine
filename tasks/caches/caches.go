package caches

import (
	"app/base/core"
	"app/base/utils"
	"app/tasks"
)

var (
	skipNAccountsRefresh int
)

func configure() {
	core.ConfigureApp()
}

func RunPackageRefresh() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	configure()
	utils.LogInfo("Refreshing package cache")
	errRefresh := RefreshPackagesCaches(nil)
	if err := Metrics().Add(); err != nil {
		utils.LogInfo("err", err, "Could not push to pushgateway")
	}
	if errRefresh != nil {
		utils.LogError("err", errRefresh, "Refresh account packages caches")
		return
	}
	utils.LogInfo("Refreshed account packages caches")
}
