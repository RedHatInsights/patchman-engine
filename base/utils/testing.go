package utils

import (
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

func SkipWithoutDB(t *testing.T) {
	if os.Getenv("USE_TESTING_DB") != "on" {
		t.Skip("testing database not used - skipping")
	}
}

func SkipWithoutPlatform(t *testing.T) {
	if os.Getenv("VMAAS_ADDRESS") == "" {
		t.Skip("testing platform instance not used - skipping")
	}
}

type TestLogHook struct {
	LogEntries []log.Entry
}

func (t *TestLogHook) Levels() []log.Level {
	return []log.Level{log.InfoLevel, log.WarnLevel}
}

func (t *TestLogHook) Fire(entry *log.Entry) error {
	t.LogEntries = append(t.LogEntries, *entry)
	return nil
}
