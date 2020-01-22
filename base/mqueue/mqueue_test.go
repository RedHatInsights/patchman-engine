package mqueue

import (
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"testing"
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
