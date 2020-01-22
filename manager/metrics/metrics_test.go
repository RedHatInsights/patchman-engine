package metrics

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSystemsCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	optOuted, notOptOuted, err := getSystemCounts()
	assert.Nil(t, err)
	assert.Equal(t, 0, optOuted)
	assert.Equal(t, 12, notOptOuted)
}

func TestAdvisoryCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	unknown, enh, bug, sec, err := getAdvisoryCounts()
	assert.Nil(t, err)
	assert.Equal(t, 0, unknown)
	assert.Equal(t, 3, enh)
	assert.Equal(t, 3, bug)
	assert.Equal(t, 2, sec)
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
