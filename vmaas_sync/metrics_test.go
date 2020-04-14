package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSystemsCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	optOuted, notOptOuted, err := getSystemCounts()
	assert.Nil(t, err)
	assert.Equal(t, 0, optOuted)
	assert.Equal(t, 12, notOptOuted)
}

func TestUploadedSystemsCounts1D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	refTime := time.Date(2018, 9, 23, 10, 0, 0, 0, time.UTC)
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime, 1)
	var nSystems int
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, 2, nSystems)
}

func TestUploadedSystemsCounts7D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	refTime := time.Date(2018, 9, 23, 10, 0, 0, 0, time.UTC)
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime, 7)
	var nSystems int
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, 5, nSystems)
}

func TestUploadedSystemsCounts30D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	refTime := time.Date(2018, 9, 23, 10, 0, 0, 0, time.UTC)
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime, 30)
	var nSystems int
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, 8, nSystems)
}

func TestUploadedSystemsCountsNoSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	refTime := time.Date(2020, 9, 23, 10, 0, 0, 0, time.UTC)
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime, 30)
	var nSystems int
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, 0, nSystems)
}

func TestUploadedSystemsCountsAllSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	refTime := time.Date(2020, 9, 23, 10, 0, 0, 0, time.UTC)
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime, -1)
	var nSystems int
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, 12, nSystems)
}

func TestAdvisoryCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	unknown, enh, bug, sec, err := getAdvisoryCounts()
	assert.Nil(t, err)
	assert.Equal(t, 0, unknown)
	assert.Equal(t, 3, enh)
	assert.Equal(t, 3, bug)
	assert.Equal(t, 3, sec)
}

func TestSystemAdvisoriesStats(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	stats, err := getSystemAdvisorieStats()
	assert.Nil(t, err)
	assert.Equal(t, 8, stats.MaxAll)
	assert.Equal(t, 3, stats.MaxEnh)
	assert.Equal(t, 3, stats.MaxBug)
	assert.Equal(t, 2, stats.MaxSec)
}
