package mqueue

import (
	"app/base/utils"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var msg = Message{Value: []byte(`{"id": "TEST-00000", "type": "delete"}`)}

func TestParseEvents(t *testing.T) {
	reached := false

	err := MakeMessageHandler(func(event PlatformEvent) error {
		assert.Equal(t, event.ID, "TEST-00000")
		assert.Equal(t, *event.Type, "delete")
		reached = true
		return nil
	})(msg)

	assert.True(t, reached, "Event handler should have been called")
	assert.NoError(t, err)
}

func TestRoundTrip(t *testing.T) {
	utils.SkipWithoutPlatform(t)
	reader := ReaderFromEnv("test")

	resChan := make(chan PlatformEvent)
	go reader.HandleMessages(MakeMessageHandler(func(event PlatformEvent) error {
		resChan <- event
		return nil
	}))
	time.Sleep(time.Second)

	writer := WriterFromEnv("test")
	eventIn := PlatformEvent{ID: "some-id"}
	assert.NoError(t, WriteEvents(context.Background(), writer, eventIn))

	select {
	case <-time.NewTimer(time.Second * 20).C:
		assert.Fail(t, "Round trip test timed out")
	case res := <-resChan:
		assert.Equal(t, eventIn, res)
	}
}

func TestRetry(t *testing.T) {
	i := 0
	handler := func(message PlatformEvent) error {
		i++
		if i < 2 {
			return errors.New("Failed")
		}
		return nil
	}

	// Without retry handler should fail
	assert.Error(t, MakeMessageHandler(handler)(msg))

	// With retry we handler should eventually succeed
	assert.NoError(t, MakeRetryingHandler(MakeMessageHandler(handler))(msg))
}
