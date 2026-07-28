package mqueue

import (
	"app/base/utils"
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var id = uuid.MustParse("99c0ffee-0000-0000-0000-0000c0ffee99")
var someid = uuid.MustParse("99c0ffee-0000-0000-0000-0000000050de")

var msg = KafkaMessage{Value: []byte(`{"id": "` + id.String() + `", "type": "delete"}`)}

func TestRoundTripKafkaGo(t *testing.T) {
	utils.SkipWithoutPlatform(t)
	reader := NewKafkaReaderFromEnv("test")
	defer reader.Close()

	var eventOut PlatformEvent
	go reader.HandleMessages(t.Context(), func(m KafkaMessage) error {
		return sonic.Unmarshal(m.Value, &eventOut)
	})

	writer := NewKafkaWriterFromEnv("test")
	eventIn := PlatformEvent{ID: someid}
	assert.NoError(t, writePlatformEvents(context.Background(), writer, eventIn))
	utils.AssertEqualWait(t, 10, func() (exp, act interface{}) {
		return eventIn.ID, eventOut.ID
	})
}

func TestSpawnReader(t *testing.T) {
	var nReaders int32
	wg := sync.WaitGroup{}
	SpawnReader(context.Background(), &wg, "", CreateCountedMockReader(&nReaders),
		func(_ KafkaMessage) error { return nil })
	wg.Wait()
	assert.Equal(t, 1, int(nReaders))
}

func TestRetry(t *testing.T) {
	i := 0
	handler := func(_ KafkaMessage) error {
		i++
		if i < 2 {
			return errors.New("Failed")
		}
		return nil
	}

	// Without retry handler should fail
	assert.Error(t, handler(msg))

	// With retry we handler should eventually succeed
	assert.NoError(t, MakeRetryingHandler(handler)(msg))
}
