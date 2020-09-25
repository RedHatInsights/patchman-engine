package listener

import (
	"app/base/mqueue"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestRunReaders(t *testing.T) {
	nReaders := 0
	wg := sync.WaitGroup{}
	wg.Add(1)
	runReaders(&wg, mqueue.CreateCountedMockReader(&nReaders))
	time.Sleep(time.Millisecond * 300)
	// it will create CONSUMER_COUNT (8) readers
	assert.Equal(t, 8, nReaders)
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
