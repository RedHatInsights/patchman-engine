package system_culling //nolint:revive,stylecheck

import (
	"app/base/core"
	"app/base/utils"
	"app/tasks"
)

var (
	deleteCulledSystemsLimit int
	enableCulledSystemDelete bool
	enableSystemStaling      bool
)

func configure() {
	core.ConfigureApp()
	deleteCulledSystemsLimit = utils.GetIntEnvOrDefault("DELETE_CULLED_SYSTEMS_LIMIT", 1000)
	enableCulledSystemDelete = utils.GetBoolEnvOrDefault("ENABLE_CULLED_SYSTEM_DELETE", true)
	enableSystemStaling = utils.GetBoolEnvOrDefault("ENABLE_SYSTEM_STALING", true)
}

func RunSystemCulling() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	configure()

	runSystemCulling()
	if err := Metrics().Add(); err != nil {
		utils.LogInfo("err", err, "Could not push to pushgateway")
	}
}
