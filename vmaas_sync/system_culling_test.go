package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var staleDate, _ = time.Parse(base.Rfc3339NoTz, "2006-01-02T15:04:05-07:00")

func TestSingleSystemStale(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var oldAffected int
	var systems []models.SystemPlatform
	var accountData []models.AdvisoryAccountData

	database.DebugWithCachesCheck("stale-trigger", func() {
		assert.NotNil(t, staleDate)
		assert.NoError(t, database.Db.Find(&accountData, "systems_affected > 1 ").Order("systems_affected DESC").Error)
		assert.NoError(t, database.Db.Find(&systems, "rh_account_id = ?", accountData[0].RhAccountID).Error)

		systems[0].StaleTimestamp = &staleDate
		systems[0].StaleWarningTimestamp = &staleDate
		assert.NoError(t, database.Db.Save(&systems[0]).Error)
		oldAffected = accountData[0].SystemsAffected
		assert.NoError(t, database.Db.Find(&accountData, "rh_account_id = ? AND advisory_id = ?",
			accountData[0].RhAccountID, accountData[0].AdvisoryID).Error)

		assert.Equal(t, oldAffected-1, accountData[0].SystemsAffected,
			"Systems affected should be decremented by one")
	})

	database.DebugWithCachesCheck("stale-trigger", func() {
		systems[0].StaleTimestamp = nil
		systems[0].StaleWarningTimestamp = nil
		systems[0].Stale = false
		assert.NoError(t, database.Db.Save(&systems[0]).Error)
		assert.NoError(t, database.Db.Find(&accountData, "rh_account_id = ? AND advisory_id = ?",
			accountData[0].RhAccountID, accountData[0].AdvisoryID).Error)

		assert.Equal(t, oldAffected, accountData[0].SystemsAffected,
			"Systems affected should be changed to match value at the start of the test case")
	})
}

// Test for making sure system culling works
func TestMarkSystemsStale(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var systems []models.SystemPlatform
	var accountData []models.AdvisoryAccountData
	assert.NotNil(t, staleDate)
	assert.NoError(t, database.Db.Find(&systems).Error)
	assert.NoError(t, database.Db.Find(&accountData).Error)
	for i := range systems {
		assert.NotEqual(t, 0, systems[i].ID)
		// Check for valid state before modifying the systems in DB
		assert.Equal(t, false, systems[i].Stale, "No systems should be stale")
		systems[i].StaleTimestamp = &staleDate
		systems[i].StaleWarningTimestamp = &staleDate
	}

	assert.True(t, len(accountData) > 0, "We should have some systems affected by advisories")
	for _, a := range accountData {
		assert.True(t, a.SystemsAffected > 0, "We should have some systems affected")
	}
	for i := range systems {
		assert.NoError(t, database.Db.Save(&systems[i]).Error)
	}
	assert.NoError(t, database.Db.Exec("select * from mark_stale_systems()").Error)

	assert.NoError(t, database.Db.Find(&systems).Error)
	for i, s := range systems {
		assert.Equal(t, true, s.Stale, "All systems should be stale")
		s.StaleTimestamp = nil
		s.StaleWarningTimestamp = nil
		s.Stale = false
		systems[i] = s
	}

	assert.NoError(t, database.Db.Find(&accountData).Error)
	sumAffected := 0
	for _, a := range accountData {
		sumAffected += a.SystemsAffected
	}
	assert.True(t, sumAffected == 0, "all advisory_data should be deleted", sumAffected)
}

func TestMarkSystemsNotStale(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var systems []models.SystemPlatform
	var accountData []models.AdvisoryAccountData

	assert.NoError(t, database.Db.Find(&systems).Error)
	for i, s := range systems {
		assert.Equal(t, true, s.Stale, "All systems should be stale at the start of the test")
		s.StaleTimestamp = nil
		s.StaleWarningTimestamp = nil
		s.Stale = false
		systems[i] = s
	}

	for i := range systems {
		assert.NoError(t, database.Db.Save(&systems[i]).Error)
	}

	assert.NoError(t, database.Db.Find(&accountData).Error)
	assert.True(t, len(accountData) > 0, "We should have some systems affected by advisories")
	for _, a := range accountData {
		assert.True(t, a.SystemsAffected > 0, "We should have some systems affected")
	}
}

func TestCullSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var systems []models.SystemPlatform
	var cnt int
	var cntAfter int
	database.DebugWithCachesCheck("delete-culled", func() {
		assert.NoError(t, database.Db.Model(&models.SystemPlatform{}).Count(&cnt).Error)

		assert.NoError(t, database.Db.Model(&models.SystemPlatform{}).Find(&systems).Error)
		systems[0].CulledTimestamp = &staleDate
		assert.NoError(t, database.Db.Model(&models.SystemPlatform{}).Save(&systems[0]).Error)

		assert.NoError(t, database.Db.Exec("select * from delete_culled_systems()").Error)
		assert.NoError(t, database.Db.Model(&models.SystemPlatform{}).Count(&cntAfter).Error)
		assert.Equal(t, cnt-1, cntAfter)
	})
}
