package metrics

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSystemsCount(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	optOuted, notOptOuted, err := getSystemCounts()
	assert.Nil(t, err)
	assert.Equal(t, 0, optOuted)
	assert.Equal(t, 12, notOptOuted)
}
