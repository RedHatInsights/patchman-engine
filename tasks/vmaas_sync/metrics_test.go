package vmaas_sync //nolint:revive,stylecheck

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
	return database.Db.Exec("UPDATE system_platform SET last_upload = last_upload + ?", timeshift).Error
}

func TestSystemsCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	timeshift := time.Since(refTime())
	assert.Nil(t, shiftSystemsLastUpload(timeshift))
	counts, err := getSystemCounts()
	assert.Nil(t, err)
	assert.Equal(t, 2, counts[systemsCntLabels{staleOff, lastUploadLast1D}])
	assert.Equal(t, 5, counts[systemsCntLabels{staleOff, lastUploadLast7D}])
	assert.Equal(t, 8, counts[systemsCntLabels{staleOff, lastUploadLast30D}])
	assert.Equal(t, 15, counts[systemsCntLabels{staleOff, lastUploadAll}])
	assert.Equal(t, 0, counts[systemsCntLabels{staleOn, lastUploadLast1D}])
	assert.Equal(t, 0, counts[systemsCntLabels{staleOn, lastUploadLast7D}])
	assert.Equal(t, 0, counts[systemsCntLabels{staleOn, lastUploadLast30D}])
	assert.Equal(t, 0, counts[systemsCntLabels{staleOn, lastUploadAll}])
	assert.Nil(t, shiftSystemsLastUpload(-timeshift))
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
	assert.Equal(t, int64(12), count)
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
