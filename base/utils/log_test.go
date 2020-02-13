package utils

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

type testHook struct {
	LogEntries []log.Entry
}

func (t *testHook) Levels() []log.Level {
	return []log.Level{log.InfoLevel, log.WarnLevel}
}

func (t *testHook) Fire(entry *log.Entry) error {
	t.LogEntries = append(t.LogEntries, *entry)
	return nil
}

func TestInitLogging(t *testing.T) {
	assert.Nil(t, os.Setenv("LOG_STYLE", "json"))
	ConfigureLogging()

	var hook = &testHook{}
	log.AddHook(hook)

	Log("num", 1, "str", "text").Info("info log")

	assert.Equal(t, 1, len(hook.LogEntries))
	entry := hook.LogEntries[0]
	assert.Equal(t, 2, len(entry.Data))
	assert.Equal(t, 1, entry.Data["num"])
	assert.Equal(t, "text", entry.Data["str"])
}

func TestOddArgsWarn(t *testing.T) {
	assert.Nil(t, os.Setenv("LOG_STYLE", "json"))
	ConfigureLogging()

	var hook = &testHook{}
	log.AddHook(hook)

	Log("num", 1, 2).Info("info log")

	assert.Equal(t, 2, len(hook.LogEntries))
	warnEntry := hook.LogEntries[0]
	assert.Equal(t, "Unable to accept odd (3) arguments count in utils.DebugLog method.", warnEntry.Message)
	infoEntry := hook.LogEntries[1]
	assert.Equal(t, "info log", infoEntry.Message)
}
