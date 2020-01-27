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
	var data []byte
	go reader.HandleMessages(func(message kafka.Message) {
		data = message.Value
	})

	msg := kafka.Message{Value: []byte("abcd")}
	writer := WriterFromEnv("test")
	assert.NoError(t, writer.WriteMessages(context.Background(), msg))
	time.Sleep(5 * time.Second)
	assert.NotNil(t, data)
	assert.Equal(t, data, msg.Value)
}
