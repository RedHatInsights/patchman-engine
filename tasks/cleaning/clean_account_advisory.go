package cleaning

import (
	"app/base/core"
	"app/base/models"
	"app/base/utils"
	"app/tasks"
	"sync"
)

func RunCleanAccountAdvisory() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	core.ConfigureApp()
	defer utils.LogPanics(true)

	var wg sync.WaitGroup
	var aadErr, aaErr error
	wg.Add(2)

	go func() {
		defer wg.Done()
		utils.LogInfo("Deleting advisory rows with 0 applicable/installable systems from advisory_account_data")
		aadErr = CleanAdvisoryAccountData()
		if aadErr != nil {
			utils.LogError("err", aadErr, "Cleaning advisory_account_data")
		}
	}()

	go func() {
		defer wg.Done()
		utils.LogInfo("Deleting advisory rows with 0 applicable/installable systems from account_advisory")
		aaErr = CleanAccountAdvisory()
		if aaErr != nil {
			utils.LogError("err", aaErr, "Cleaning account_advisory")
		}
	}()

	wg.Wait()
	if aadErr != nil || aaErr != nil {
		utils.LogWarn("RunCleanAccountAdvisory task completed with errors")
	} else {
		utils.LogInfo("RunCleanAccountAdvisory task performed successfully")
	}
}

func CleanAdvisoryAccountData() error {
	tx := tasks.CancelableDB().Begin()
	defer tx.Rollback()

	result := tx.Delete(&models.AdvisoryAccountData{}, "systems_installable <= 0 AND systems_applicable <= 0")
	if result.Error != nil {
		return result.Error
	}

	tx.Commit()
	utils.LogInfo("nDeleted", result.RowsAffected, "advisory_account_data cleaned successfully")
	return nil
}

func CleanAccountAdvisory() error {
	tx := tasks.CancelableDB().Begin()
	defer tx.Rollback()

	result := tx.Delete(&models.AccountAdvisory{}, "systems_installable <= 0 AND systems_applicable <= 0")
	if result.Error != nil {
		return result.Error
	}

	tx.Commit()
	utils.LogInfo("nDeleted", result.RowsAffected, "account_advisory cleaned successfully")
	return nil
}
