package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/core"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableSizes(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	tableSizes := getTableSizes()
	uniqueTables := map[string]bool{}
	for _, item := range tableSizes {
		uniqueTables[item.Key] = true
	}
	assert.Equal(t, 195, len(tableSizes))
	assert.Equal(t, 195, len(uniqueTables))
	assert.True(t, uniqueTables["system_platform"]) // check whether table names were loaded
	assert.True(t, uniqueTables["package"])
	assert.True(t, uniqueTables["repo"])
}

func TestDatabaseSize(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	databaseSize := getDatabaseSize()

	assert.Equal(t, 1, len(databaseSize))
	assert.Equal(t, "database", databaseSize[0].Key)
	assert.Greater(t, databaseSize[0].Value, 0.0)
}

func TestDatabaseProcCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	processesInfo := getDatabaseProcesses()

	assert.Less(t, 0, len(processesInfo))
	ndash := 0
	for _, info := range processesInfo {
		if info.Key == "-" {
			ndash++
		}
	}
	assert.Less(t, 0, ndash)
}
