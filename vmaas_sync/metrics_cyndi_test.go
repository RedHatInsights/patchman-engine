package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCyndiSystemsCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	counts, err := getCyndiCounts(refTime())
	assert.Nil(t, err)
	assert.Equal(t, 16, counts[lastUploadLast1D])
	assert.Equal(t, 16, counts[lastUploadAll])
}

func TestUploadedCyndiSystemsCounts1D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Table("inventory.hosts")
	systemsQuery = updateCyndiQueryLastUpload(systemsQuery, refTime(), 1)
	var nSystems int64
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, int64(16), nSystems)
}

func TestUploadedCyndiSystemsCounts7D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Table("inventory.hosts")
	systemsQuery = updateCyndiQueryLastUpload(systemsQuery, refTime(), 7)
	var nSystems int64
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, int64(16), nSystems)
}

func TestUploadedCyndiSystemsCounts30D(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Table("inventory.hosts")
	systemsQuery = updateCyndiQueryLastUpload(systemsQuery, refTime(), 30)
	var nSystems int64
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, int64(16), nSystems)
}

func TestUploadedCyndiSystemsCountsNoSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Table("inventory.hosts")
	refTime := time.Date(2020, 9, 23, 10, 0, 0, 0, time.UTC)
	systemsQuery = updateCyndiQueryLastUpload(systemsQuery, refTime, 30)
	var nSystems int64
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, int64(0), nSystems)
}

func TestUploadedCyndiSystemsCountsAllSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemsQuery := database.Db.Table("inventory.hosts")
	systemsQuery = updateCyndiQueryLastUpload(systemsQuery, refTime(), -1)
	var nSystems int64
	assert.Nil(t, systemsQuery.Count(&nSystems).Error)
	assert.Equal(t, int64(16), nSystems)
}

func TestCyndiTags(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	cyndiData, err := getCyndiData()
	assert.Nil(t, err)
	assert.Equal(t, int64(16), cyndiData.SystemsCount)
	assert.Equal(t, int64(3), cyndiData.UniqueTags)
	assert.Equal(t, int64(14), cyndiData.SapCount)
	assert.Equal(t, int64(15), cyndiData.SystemsWithTags)
}
