package system_culling //nolint:revive,stylecheck

import (
	"app/base/core"
	"app/base/utils"
	"app/tasks"
)

var (
	deleteCulledSystemsLimit int
)

func configure() {
	core.ConfigureApp()
	deleteCulledSystemsLimit = utils.GetIntEnvOrDefault("DELETE_CULLED_SYSTEMS_LIMIT", 1000)
}

func RunSystemCulling() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	configure()

	runSystemCulling()
	if err := Metrics().Add(); err != nil {
		utils.LogInfo("err", err, "Could not push to pushgateway")
	}
}
