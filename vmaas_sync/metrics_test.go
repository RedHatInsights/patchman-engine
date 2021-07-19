package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
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

	systemsQuery := database.Db.Model(models.SystemPlatform{}).Session(&gorm.Session{PrepareStmt: true})
	var optOuted int64
	var notOptOuted int64
	assert.Nil(t, updateSystemsQueryStale(systemsQuery, true).Count(&optOuted).Error)
	assert.Nil(t, updateSystemsQueryStale(systemsQuery, false).Count(&notOptOuted).Error)
	assert.Equal(t, int64(0), optOuted)
	assert.Equal(t, int64(14), notOptOuted)
}

func TestUploadedSystemsCounts1D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime(), 1)
	var nSystems int64
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, int64(2), nSystems)
}

func TestUploadedSystemsCounts7D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime(), 7)
	var nSystems int64
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, int64(5), nSystems)
}

func TestUploadedSystemsCounts30D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime(), 30)
	var nSystems int64
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, int64(8), nSystems)
}

func TestUploadedSystemsCountsNoSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	refTime := time.Date(2020, 9, 23, 10, 0, 0, 0, time.UTC)
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime, 30)
	var nSystems int64
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, int64(1), nSystems)
}

func TestUploadedSystemsCountsAllSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Model(models.SystemPlatform{})
	systemsQuery = updateSystemsQueryLastUpload(systemsQuery, refTime(), -1)
	var nSystems int64
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, int64(14), nSystems)
}

func TestAdvisoryCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	unknown, enh, bug, sec, err := getAdvisoryCounts()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), unknown)
	assert.Equal(t, int64(3), enh)
	assert.Equal(t, int64(3), bug)
	assert.Equal(t, int64(3), sec)
}

func TestPackageCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	database.DeleteNewlyAddedPackages(t)

	count, err := getPackageCounts()
	assert.Nil(t, err)
	assert.Equal(t, int64(12), count)
}

func TestPackageNameCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	count, err := getPackageNameCounts()
	assert.Nil(t, err)
	assert.Equal(t, int64(10), count)
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
