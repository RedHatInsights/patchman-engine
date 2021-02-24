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

func refTime() time.Time {
	return time.Date(2018, 9, 23, 10, 0, 0, 0, time.UTC)
}

func TestSystemsCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	counts, err := getSystemCounts(refTime())
	assert.Nil(t, err)
	assert.Equal(t, 2, counts[systemsCntLabels{staleOff, lastUploadLast1D}])
	assert.Equal(t, 14, counts[systemsCntLabels{staleOff, lastUploadAll}])
	assert.Equal(t, 0, counts[systemsCntLabels{staleOn, lastUploadAll}])
}

func TestSystemsCountsStale(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	var stale int
	var notStale int
	assert.Nil(t, updateSystemsQueryStale(systemsQuery, true).Count(&stale).Error)
	assert.Nil(t, updateSystemsQueryStale(systemsQuery, false).Count(&notStale).Error)
	assert.Equal(t, 0, stale)
	assert.Equal(t, 14, notStale)
}

func TestUploadedSystemsCounts1D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime(), 1)
	var nSystems int
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, 2, nSystems)
}

func TestUploadedSystemsCounts7D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime(), 7)
	var nSystems int
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, 5, nSystems)
}

func TestUploadedSystemsCounts30D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime(), 30)
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
	assert.Equal(t, 1, nSystems)
}

func TestUploadedSystemsCountsAllSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime(), -1)
	var nSystems int
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, 14, nSystems)
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
