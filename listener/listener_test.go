package listener

import (
	"app/base/mqueue"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRunReaders(t *testing.T) {
	nReaders := 0
	runReaders(mqueue.CreateCountedMockReader(&nReaders))
	time.Sleep(time.Millisecond * 300)
	assert.Equal(t, 1, nReaders)
}

func TestLoadValidReporters(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	configure()

	reporter := loadValidReporters()
	assert.Equal(t, 3, len(reporter))
	assert.Equal(t, 1, reporter["puptoo"])
	assert.Equal(t, 2, reporter["rhsm-conduit"])
	assert.Equal(t, 3, reporter["yupana"])
}
