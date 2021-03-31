package listener

import (
	"app/base/mqueue"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestRunReaders(t *testing.T) {
	nReaders := 0
	wg := sync.WaitGroup{}
	wg.Add(1)
	runReaders(&wg, mqueue.CreateCountedMockReader(&nReaders))
	nReadersExpected := 8
	utils.AssertEqualWait(t, 10, func() (exp, act interface{}) {
		return nReadersExpected, nReaders
	})
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
