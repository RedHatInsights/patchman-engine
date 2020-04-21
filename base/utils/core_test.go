package utils

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRecoverAndLogPanics(t *testing.T) {
	ConfigureLogging()

	logHook := NewTestLogHook()
	log.AddHook(logHook)

	func() {
		defer LogPanics(false)
		panic("We crashed")
	}()

	assert.Equal(t, 1, len(logHook.LogEntries))
}
