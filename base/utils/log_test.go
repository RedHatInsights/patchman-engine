package utils

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInitLogging(t *testing.T) {
	assert.Nil(t, os.Setenv("LOG_STYLE", "json"))
	ConfigureLogging()

	var hook = NewTestLogHook()
	log.AddHook(hook)

	LogInfo("num", 1, "str", "text", "info log")

	assert.Equal(t, 1, len(hook.LogEntries))
	entry := hook.LogEntries[0]
	assert.Equal(t, 2, len(entry.Data))
	assert.Equal(t, 1, entry.Data["num"])
	assert.Equal(t, "text", entry.Data["str"])
	assert.Equal(t, "info log", entry.Message)
}

func TestEvenArgs(t *testing.T) {
	assert.Nil(t, os.Setenv("LOG_STYLE", "json"))
	ConfigureLogging()

	var hook = NewTestLogHook()
	log.AddHook(hook)

	LogInfo("num", 1, "str", "text")

	assert.Equal(t, 1, len(hook.LogEntries))
	entry := hook.LogEntries[0]
	assert.Equal(t, 2, len(entry.Data))
	assert.Equal(t, 1, entry.Data["num"])
	assert.Equal(t, "text", entry.Data["str"])
}
