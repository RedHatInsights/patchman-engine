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
	utils.Log().Info("Refreshing advisory cache")
	RefreshAdvisoryCaches()
}
