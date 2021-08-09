package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/core"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCyndiMetrics(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	tagCounts, systemCounts, err := getCyndiData()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), systemCounts[lastUploadLast1D])
	assert.Equal(t, int64(0), systemCounts[lastUploadLast7D])
	assert.Equal(t, int64(0), systemCounts[lastUploadLast30D])
	assert.Equal(t, int64(16), systemCounts[lastUploadAll])
	assert.Equal(t, int64(16), tagCounts[allSystemCount])
	assert.Equal(t, int64(14), tagCounts[systemsSapSystemCount])
	assert.Equal(t, int64(15), tagCounts[systemsWithTagsCount])
}
