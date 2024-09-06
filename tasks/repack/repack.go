package repack

import (
	"app/base/core"
	"app/base/utils"
	"app/tasks"
	"fmt"
)

var pgRepackArgs = []string{
	"pg_repack", "--no-superuser-check",
	"-d", utils.CoreCfg.DBName,
	"-h", utils.CoreCfg.DBHost,
	"-p", fmt.Sprintf("%d", utils.CoreCfg.DBPort),
	"-U", utils.CoreCfg.DBUser,
}

func configure() {
	core.ConfigureApp()
}

func Repack(table string) error {
	// TODO
	return nil
}

func RunRepack() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	utils.LogInfo("Starting repack job")
	configure()

	var table_name string = "system_package2" // FIXME: use env variable maybe

	err := Repack(table_name)
	if err != nil {
		utils.LogError("err", err.Error(), fmt.Sprintf("Repack table %s", table_name))
		return
	}
	utils.LogInfo(fmt.Sprintf("Repacked table %s", table_name))
}
