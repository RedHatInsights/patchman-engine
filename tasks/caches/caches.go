package caches

import (
	"app/base/core"
	"app/base/utils"
	"app/tasks"
)

var (
	enableRefreshAdvisoryCaches bool
)

func configure() {
	core.ConfigureApp()
	enableRefreshAdvisoryCaches = utils.GetBoolEnvOrDefault("ENABLE_REFRESH_ADVISORY_CACHES", false)
}

func RunAdvisoryRefresh() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	configure()
	utils.LogInfo("Refreshing advisory cache")
	RefreshAdvisoryCaches()
}

func RunPackageRefresh() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	configure()
	utils.LogInfo("Refreshing package cache")
	if err := RefreshPackagesCaches(nil); err != nil {
		utils.LogError("err", err.Error(), "Refresh account packages caches")
		return
	}
	utils.LogInfo("Refreshed account packages caches")
}
