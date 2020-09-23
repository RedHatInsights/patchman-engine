package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTableSizes(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	tableSizes := getTableSizes()

	assert.Equal(t, 19, len(tableSizes))
}

func TestDatabaseSize(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	itemSize := getDatabaseSize()

	assert.Equal(t, 1, len(itemSize))
	assert.Equal(t, "database", itemSize[0].Name)
	assert.Greater(t, itemSize[0].Size, 0.0)
}
