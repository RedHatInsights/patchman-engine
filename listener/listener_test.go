package listener

import (
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadValidReporters(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	configure()

	reporter := loadValidReporters()
	assert.Equal(t, 4, len(reporter))
	assert.Equal(t, 1, reporter["puptoo"])
	assert.Equal(t, 2, reporter["rhsm-conduit"])
	assert.Equal(t, 3, reporter["yupana"])
	assert.Equal(t, 4, reporter["rhsm-system-profile-bridge"])
}
