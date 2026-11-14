package mqueue

import (
	"app/base"
	"app/base/utils"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
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
	go reader.HandleMessages(t.Context(), 1, 0, func(_ context.Context, m []KafkaMessage) error {
		return sonic.Unmarshal(m[0].Value, &eventOut)
	})

	writer := NewKafkaWriterFromEnv("test")
	eventIn := PlatformEvent{ID: someid}
	assert.NoError(t, writePlatformEvents(base.Context, writer, eventIn))
	utils.AssertEqualWait(t, 10, func() (exp, act interface{}) {
		return eventIn.ID, eventOut.ID
	})
}

func TestKeyedKafkaWriterRoutesSameAccountToSamePartition(t *testing.T) {
	w := NewKeyedKafkaWriterFromEnv("patchman.advisory.update").(*kafkaGoWriterImpl)
	assert.IsType(t, &kafka.Hash{}, w.Balancer)

	partitions := []int{0, 1, 2, 3, 4, 5}
	msg := kafka.Message{Key: []byte("42")}
	first := w.Balancer.Balance(msg, partitions...)
	second := w.Balancer.Balance(msg, partitions...)
	assert.Equal(t, first, second)
}

func TestSpawnReader(t *testing.T) {
	var nReaders int32
	wg := sync.WaitGroup{}
	SpawnReader(base.Context, &wg, "", CreateCountedMockReader(&nReaders), 1, 0,
		func(_ context.Context, _ []KafkaMessage) error { return nil })
	wg.Wait()
	assert.Equal(t, 1, int(nReaders))
}

func TestSpawnReaderBatch(t *testing.T) {
	var nReaders int32
	wg := sync.WaitGroup{}
	SpawnReader(base.Context, &wg, "", CreateCountedMockReader(&nReaders), 2, time.Millisecond,
		func(context.Context, []KafkaMessage) error { return nil })
	wg.Wait()
	assert.Equal(t, 1, int(nReaders))
}

func TestSpawnReaderPanicsWithoutHandler(t *testing.T) {
	assert.PanicsWithValue(t, "reader loop has no handler", func() {
		SpawnReader(base.Context, &sync.WaitGroup{}, "", CreateCountedMockReader(new(int32)), 1, 0, nil)
	})
}

func TestRetryBatchFatalThenSuccess(t *testing.T) {
	i := 0
	handler := func(_ context.Context, _ []KafkaMessage) error {
		i++
		if i < 3 {
			return base.ErrFatal
		}
		return nil
	}

	assert.NoError(t, MakeRetryingHandler(handler)(base.Context, []KafkaMessage{msg}))
	assert.Equal(t, 3, i)
}

func TestRetryBatchAlwaysFatal(t *testing.T) {
	handler := func(_ context.Context, _ []KafkaMessage) error {
		return base.ErrFatal
	}

	err := MakeRetryingHandler(handler)(base.Context, []KafkaMessage{msg})
	assert.ErrorIs(t, err, base.ErrFatal)
}

func TestWriteMessagesInjectsTraceparent(t *testing.T) {
	// WriteMessages requires a Kafka broker; inject coverage lives in
	// telemetry.Inject / TestProducerContextLinksOriginalsAndInjectsOwnTraceparent.
	t.Skip("no broker-free WriteMessages path; covered by telemetry.Inject tests")
}
