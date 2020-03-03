package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var staleDate, _ = time.Parse(base.Rfc3339NoTz, "2006-01-02T15:04:05-07:00")

// Test for making sure system culling works
func TestMarkSystemsStale(t *testing.T) {
	utils.SkipWithoutDB(t)
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

	assert.NoError(t, database.Db.Exec("select * from mark_stale_systems()").Error)
}
