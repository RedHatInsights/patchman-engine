package caches

import (
	"app/base/core"
	"app/base/utils"
	"app/tasks"
)

var (
	enableRefreshAdvisoryCaches bool
	skipNAccountsRefresh        int
)

func configure() {
	core.ConfigureApp()
	enableRefreshAdvisoryCaches = utils.GetBoolEnvOrDefault("ENABLE_REFRESH_ADVISORY_CACHES", false)
	skipNAccountsRefresh = utils.GetIntEnvOrDefault("SKIP_N_ACCOUNTS_REFRESH", 0)
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
	errRefresh := RefreshPackagesCaches(nil)
	if err := Metrics().Add(); err != nil {
		utils.LogInfo("err", err, "Could not push to pushgateway")
	}
	if errRefresh != nil {
		utils.LogError("err", errRefresh.Error(), "Refresh account packages caches")
		return
	}
	utils.LogInfo("Refreshed account packages caches")
}
