package mqueue

import (
	"app/base/core"
	"app/base/utils"
	"context"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestParseEvents(t *testing.T) {
	msg := kafka.Message{Value: []byte(`{"id": "TEST-00000", "type": "delete"}`)}

	reached := false

	makeKafkaHandler(func(event PlatformEvent) {
		assert.Equal(t, event.ID, "TEST-00000")
		assert.Equal(t, *event.Type, "delete")
		reached = true
	})(msg)

	assert.True(t, reached, "Event handler should have been called")
}

func TestRoundTrip(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()

	reader := ReaderFromEnv("test")
	var eventOut PlatformEvent
	go reader.HandleEvents(func(event PlatformEvent) {
		eventOut = event
	})

	writer := WriterFromEnv("test")
	eventIn := PlatformEvent{ID: "some-id"}
	assert.NoError(t, (*writer).WriteEvent(context.Background(), eventIn))
	time.Sleep(8 * time.Second)
	assert.Equal(t, eventIn, eventOut)
}
