package system_culling //nolint:revive,stylecheck

import (
	"app/base/core"
	"app/base/utils"
	"app/tasks"
)

func configure() {
	core.ConfigureApp()
}

func RunSystemCulling() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	configure()

	runSystemCulling()
	if err := Metrics().Add(); err != nil {
		utils.LogInfo("err", err, "Could not push to pushgateway")
	}
}
