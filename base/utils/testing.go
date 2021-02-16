package utils

import (
	"context"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
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
	LogEntries    []log.Entry
	LevelsToStore []log.Level
}

func (t *TestLogHook) Levels() []log.Level {
	return t.LevelsToStore
}

func (t *TestLogHook) Fire(entry *log.Entry) error {
	t.LogEntries = append(t.LogEntries, *entry)
	return nil
}

func NewTestLogHook(levelsToStore ...log.Level) *TestLogHook {
	if len(levelsToStore) == 0 {
		levelsToStore = []log.Level{log.PanicLevel, log.FatalLevel, log.ErrorLevel, log.WarnLevel, log.InfoLevel,
			log.DebugLevel, log.TraceLevel}
	}
	return &TestLogHook{LevelsToStore: levelsToStore}
}

type MockKafkaWriter struct {
	Messages []kafka.Message
}

func (t *MockKafkaWriter) WriteMessages(_ context.Context, ev ...kafka.Message) error {
	t.Messages = append(t.Messages, ev...)
	return nil
}

func AssertWait(t *testing.T, timeoutSeconds int, funToAssert func() bool) {
	for i := 0; i < timeoutSeconds*10; i++ {
		time.Sleep(time.Millisecond * 100)
		if funToAssert() {
			break
		}
	}
	assert.True(t, funToAssert())
}
