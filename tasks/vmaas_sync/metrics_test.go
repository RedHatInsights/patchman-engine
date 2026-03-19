package vmaas_sync

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func refTime() time.Time {
	return time.Date(2018, 9, 23, 10, 0, 0, 0, time.UTC)
}

func shiftSystemsLastUpload(timeshift time.Duration) error {
	return database.DB.Exec("UPDATE system_platform SET last_upload = last_upload + ?", timeshift).Error
}

func TestSystemsCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	timeshift := time.Since(refTime())
	assert.Nil(t, shiftSystemsLastUpload(timeshift))
	counts, err := getSystemCounts()
	assert.Nil(t, err)
	assert.Equal(t, 3, counts[systemsCntLabels{staleOff, lastUploadLast1D}])
	assert.Equal(t, 6, counts[systemsCntLabels{staleOff, lastUploadLast7D}])
	assert.Equal(t, 9, counts[systemsCntLabels{staleOff, lastUploadLast30D}])
	assert.Equal(t, 18, counts[systemsCntLabels{staleOff, lastUploadAll}])
	assert.Equal(t, 0, counts[systemsCntLabels{staleOn, lastUploadLast1D}])
	assert.Equal(t, 0, counts[systemsCntLabels{staleOn, lastUploadLast7D}])
	assert.Equal(t, 0, counts[systemsCntLabels{staleOn, lastUploadLast30D}])
	assert.Equal(t, 0, counts[systemsCntLabels{staleOn, lastUploadAll}])
	assert.Nil(t, shiftSystemsLastUpload(-timeshift))
}

func TestGetSystemInventoryData(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	tagStats, systemStats, err := getSystemInventoryData()
	assert.Nil(t, err)
	assert.NotNil(t, tagStats)
	assert.NotNil(t, systemStats)

	// Tag stats: all systems, SAP systems, systems with tags (from test data: 18 total, 15 SAP, 16 with tags)
	assert.Equal(t, int64(18), tagStats[allSystemCount])
	assert.Equal(t, int64(15), tagStats[systemsSapSystemCount])
	assert.Equal(t, int64(16), tagStats[systemsWithTagsCount])

	// System stats: updated in last 1D/7D/30D and total
	assert.GreaterOrEqual(t, systemStats[lastUploadLast1D], int64(0))
	assert.GreaterOrEqual(t, systemStats[lastUploadLast7D], int64(0))
	assert.GreaterOrEqual(t, systemStats[lastUploadLast30D], int64(0))
	assert.Equal(t, int64(18), systemStats[lastUploadAll])
}

func TestAdvisoryCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	other, enh, bug, sec, err := getAdvisoryCounts()
	assert.Nil(t, err)
	assert.Equal(t, int64(4), other)
	assert.Equal(t, int64(3), enh)
	assert.Equal(t, int64(3), bug)
	assert.Equal(t, int64(4), sec)
}

func TestPackageCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	database.DeleteNewlyAddedPackages(t)

	count, err := getPackageCounts()
	assert.Nil(t, err)
	assert.Equal(t, int64(14), count)
}

func TestPackageNameCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	count, err := getPackageNameCounts()
	assert.Nil(t, err)
	assert.Equal(t, int64(12), count)
}

func TestSystemAdvisoriesStats(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	stats, err := getSystemAdvisorieStats()
	assert.Nil(t, err)
	assert.Equal(t, 5, stats.MaxAll)
	assert.Equal(t, 1, stats.MaxEnh)
	assert.Equal(t, 2, stats.MaxBug)
	assert.Equal(t, 2, stats.MaxSec)
}
