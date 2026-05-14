package caches

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testWorkspace = "00000000-0000-0000-0000-000000000001"

func TestRefreshAccountAdvisoryCaches(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	workspace := testWorkspace

	// populate account_advisory using backfill
	assert.Nil(t, database.DB.Exec("SELECT backfill_account_advisory(1)").Error)

	// capture correct counts before corrupting
	countAdv1 := database.PluckInt(database.DB.Table("account_advisory").
		Where("advisory_id = 1 AND rh_account_id = 1 AND workspace_id = ?", workspace),
		"systems_installable")
	countAdv2 := database.PluckInt(database.DB.Table("account_advisory").
		Where("advisory_id = 2 AND rh_account_id = 1 AND workspace_id = ?", workspace),
		"systems_installable")

	// set wrong counts
	assert.Nil(t, database.DB.Model(&models.AccountAdvisory{}).
		Where("advisory_id = 1 AND rh_account_id = 1 AND workspace_id = ?", workspace).
		Update("systems_installable", 99).Error)
	assert.Nil(t, database.DB.Model(&models.AccountAdvisory{}).
		Where("advisory_id = 2 AND rh_account_id = 1 AND workspace_id = ?", workspace).
		Update("systems_installable", 77).Error)

	// refresh should correct them
	assert.Nil(t, database.DB.Exec("SELECT refresh_account_advisory_caches(NULL, 1)").Error)

	assert.Equal(t, countAdv1, database.PluckInt(database.DB.Table("account_advisory").
		Where("advisory_id = 1 AND rh_account_id = 1 AND workspace_id = ?", workspace),
		"systems_installable"))
	assert.Equal(t, countAdv2, database.PluckInt(database.DB.Table("account_advisory").
		Where("advisory_id = 2 AND rh_account_id = 1 AND workspace_id = ?", workspace),
		"systems_installable"))

	// cleanup
	database.DeleteAccountAdvisoryByAccount(t, 1)
}

func TestRefreshAccountAdvisoryCachesRemovesOrphanedRows(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	workspace := testWorkspace
	assert.Nil(t, database.DB.Exec("SELECT backfill_account_advisory(1)").Error)

	// mark all systems in this workspace as stale
	assert.Nil(t, database.DB.Exec(
		"UPDATE system_inventory SET stale = true WHERE rh_account_id = 1 AND workspace_id = ?",
		workspace).Error)

	// refresh should remove rows for this workspace since no non-stale systems remain
	assert.Nil(t, database.DB.Exec("SELECT refresh_account_advisory_caches(NULL, 1)").Error)

	var count int64
	assert.Nil(t, database.DB.Table("account_advisory").
		Where("rh_account_id = 1 AND workspace_id = ?", workspace).
		Count(&count).Error)
	assert.Equal(t, int64(0), count)

	// restore systems to non-stale
	assert.Nil(t, database.DB.Exec(
		"UPDATE system_inventory SET stale = false WHERE rh_account_id = 1 AND workspace_id = ?",
		workspace).Error)

	// cleanup
	database.DeleteAccountAdvisoryByAccount(t, 1)
}

func TestTriggerAccountAdvisoryStaleChange(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	workspace := testWorkspace

	// populate account_advisory using backfill
	assert.Nil(t, database.DB.Exec("SELECT backfill_account_advisory(1)").Error)

	advCount := database.PluckInt(database.DB.Table("account_advisory").
		Where("advisory_id = 1 AND rh_account_id = 1 AND workspace_id = ?", workspace),
		"systems_installable")

	// mark system 1 as stale — trigger should decrement by 1
	assert.Nil(t, database.DB.Exec(
		"UPDATE system_inventory SET stale = true WHERE id = 1 AND rh_account_id = 1").Error)

	after := database.PluckInt(database.DB.Table("account_advisory").
		Where("advisory_id = 1 AND rh_account_id = 1 AND workspace_id = ?", workspace),
		"systems_installable")
	assert.Equal(t, advCount-1, after)

	// un-stale — trigger should increment back
	assert.Nil(t, database.DB.Exec(
		"UPDATE system_inventory SET stale = false WHERE id = 1 AND rh_account_id = 1").Error)

	restored := database.PluckInt(database.DB.Table("account_advisory").
		Where("advisory_id = 1 AND rh_account_id = 1 AND workspace_id = ?", workspace),
		"systems_installable")
	assert.Equal(t, advCount, restored)

	// cleanup
	database.DeleteAccountAdvisoryByAccount(t, 1)
}

func TestTriggerAccountAdvisoryWorkspaceChange(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	oldWorkspace := testWorkspace
	newWorkspace := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	assert.Nil(t, database.DB.Exec("SELECT backfill_account_advisory(1)").Error)

	advCount := database.PluckInt(database.DB.Table("account_advisory").
		Where("advisory_id = 1 AND rh_account_id = 1 AND workspace_id = ?", oldWorkspace),
		"systems_installable")

	// move system 1 to a new workspace — trigger should decrement old, increment new
	assert.Nil(t, database.DB.Exec(
		"UPDATE system_inventory SET workspace_id = ? WHERE id = 1 AND rh_account_id = 1",
		newWorkspace).Error)

	oldCount := database.PluckInt(database.DB.Table("account_advisory").
		Where("advisory_id = 1 AND rh_account_id = 1 AND workspace_id = ?", oldWorkspace),
		"systems_installable")
	assert.Equal(t, advCount-1, oldCount)

	newCount := database.PluckInt(database.DB.Table("account_advisory").
		Where("advisory_id = 1 AND rh_account_id = 1 AND workspace_id = ?", newWorkspace),
		"systems_installable")
	assert.Equal(t, 1, newCount)

	// restore system 1 to original workspace
	assert.Nil(t, database.DB.Exec(
		"UPDATE system_inventory SET workspace_id = ? WHERE id = 1 AND rh_account_id = 1",
		oldWorkspace).Error)

	// cleanup
	database.DeleteAccountAdvisoryByAccount(t, 1)
}
