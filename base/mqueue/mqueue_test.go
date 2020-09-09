package mqueue

import (
	"app/base/utils"
	"context"
	"errors"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"os"
	"sync"
	"testing"
	"time"
)

var msg = kafka.Message{Value: []byte(`{"id": "TEST-00000", "type": "delete"}`)}

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

	var eventOut PlatformEvent
	go reader.HandleMessages(MakeMessageHandler(func(event PlatformEvent) error {
		eventOut = event
		return nil
	}))

	writer := WriterFromEnv("test")
	eventIn := PlatformEvent{ID: "some-id"}
	assert.NoError(t, WriteEvents(context.Background(), writer, eventIn))
	time.Sleep(8 * time.Second)
	assert.Equal(t, eventIn, eventOut)
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

func TestTimeout(t *testing.T) {
	wg := sync.WaitGroup{}

	handler := func(message kafka.Message) error {
		return nil
	}
	assert.NoError(t, os.Setenv("KAFKA_READER_TIMEOUT_SECONDS", "1"))
	assert.Panics(t, func() {
		SpawnTimedReaderGroup(&wg, 1, "", CreateBlockingReader, handler)
		time.Sleep(time.Second * 5)
	})
}
