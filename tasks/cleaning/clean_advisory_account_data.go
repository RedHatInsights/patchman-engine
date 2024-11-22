package cleaning

import (
	"app/base/core"
	"app/base/models"
	"app/base/utils"
	"app/tasks"
)

func RunCleanAdvisoryAccountData() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	core.ConfigureApp()
	defer utils.LogPanics(true)
	utils.LogInfo("Deleting advisory rows with 0 applicable systems from advisory_account_data")

	if err := CleanAdvisoryAccountData(); err != nil {
		utils.LogError("err", err.Error(), "Cleaning advisory account data")
		return
	}
	utils.LogInfo("CleanAdvisoryAccountData task performed successfully")
}

func CleanAdvisoryAccountData() error {
	tx := tasks.CancelableDB().Begin()
	defer tx.Rollback()

	err := tx.Delete(&models.AdvisoryAccountData{}, "systems_installable <= 0 AND systems_applicable <= 0").Error
	if err != nil {
		return err
	}

	tx.Commit()
	return nil
}
