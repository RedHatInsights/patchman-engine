package repack

import (
	"app/base/core"
	"app/base/utils"
	"app/tasks"
	"fmt"
	"os"
	"os/exec"
)

var pgRepackArgs = []string{
	"--no-superuser-check",
	"--no-password",
	"-d", utils.CoreCfg.DBName,
	"-h", utils.CoreCfg.DBHost,
	"-p", fmt.Sprintf("%d", utils.CoreCfg.DBPort),
	"-U", utils.CoreCfg.DBUser,
}

func configure() {
	core.ConfigureApp()
}

// GetCmd returns command that calls gp_repack with table. Table must be a partitioned table.
// Args are appended to the necessary pgRepackArgs. Stdout and stderr of the subprocess are redirected to host.
func getCmd(table string, args ...string) *exec.Cmd {
	fullArgs := pgRepackArgs
	fullArgs = append(fullArgs, "-I", table)
	fullArgs = append(fullArgs, args...)
	cmd := exec.Command("pg_repack", fullArgs...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", utils.CoreCfg.DBPassword))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// Repack runs pg_repack reindex with table. If columns are provided, cluster by these columns is executed as well.
// Table must be a partitioned table.
func Repack(table string, columns string) error {
	reindexCmd := getCmd(table, "-x")
	err := reindexCmd.Run()
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		utils.LogWarn("no columns provided, skipping repack clustering")
		return nil
	}
	clusterCmd := getCmd(table, "-o", columns)
	err = clusterCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// RunRepack wraps Repack call for a job.
func RunRepack() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	utils.LogInfo("Starting repack job")
	configure()

	TABLES := map[string]string{
		"system_package2":   "rh_account_id,system_id",
		"system_platform":   "rh_account_id,id,inventory_id",
		"system_advisories": "rh_account_id,system_id",
	}

	for table, columns := range TABLES {
		err := Repack(table, columns)
		if err != nil {
			utils.LogError("err", err.Error(), fmt.Sprintf("Failed to repack table %s", table))
			continue
		}
		utils.LogInfo(fmt.Sprintf("Successfully repacked table %s", table))
	}
}
