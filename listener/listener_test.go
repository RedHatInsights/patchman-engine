package listener

import (
	"app/base/mqueue"
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
	// it will create CONSUMER_COUNT (8) * topics (2) = readers (16)
	assert.Equal(t, 16, nReaders)
}
