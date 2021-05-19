package mqueue

import (
	"app/base/utils"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

const id = "99c0ffee-0000-0000-0000-0000c0ffee99"
const someid = "99c0ffee-0000-0000-0000-0000000050de"

var msg = KafkaMessage{Value: []byte(`{"id": "` + id + `", "type": "delete"}`)}

func TestParseEvents(t *testing.T) {
	reached := false

	err := MakeMessageHandler(func(event PlatformEvent) error {
		assert.Equal(t, event.ID, id)
		assert.Equal(t, *event.Type, "delete")
		reached = true
		return nil
	})(msg)

	assert.True(t, reached, "Event handler should have been called")
	assert.NoError(t, err)
}

func TestRoundTripKafkaGo(t *testing.T) {
	utils.SkipWithoutPlatform(t)
	reader := newKafkaGoReaderFromEnv("test")

	var eventOut PlatformEvent
	go reader.HandleMessages(MakeMessageHandler(func(event PlatformEvent) error {
		eventOut = event
		return nil
	}))

	writer := newKafkaGoWriterFromEnv("test")
	eventIn := PlatformEvent{ID: someid}
	assert.NoError(t, WriteEvents(context.Background(), writer, eventIn))
	utils.AssertEqualWait(t, 8, func() (exp, act interface{}) {
		return eventIn.ID, eventOut.ID
	})
}

func TestSpawnReader(t *testing.T) {
	nReaders := 0
	wg := sync.WaitGroup{}
	SpawnReader(&wg, "", CreateCountedMockReader(&nReaders),
		MakeMessageHandler(func(event PlatformEvent) error { return nil }))
	wg.Wait()
	assert.Equal(t, 1, nReaders)
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
