package mqueue

import (
	"app/base/telemetry"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestAsBatchHandler(t *testing.T) {
	var got []KafkaMessage
	err := AsBatchHandler(func(_ context.Context, m KafkaMessage) error {
		got = append(got, m)
		return nil
	})(context.Background(), []KafkaMessage{{Value: []byte("a")}})
	assert.NoError(t, err)
	assert.Equal(t, []KafkaMessage{{Value: []byte("a")}}, got)

	got = nil
	err = AsBatchHandler(func(_ context.Context, m KafkaMessage) error {
		got = append(got, m)
		return nil
	})(context.Background(), []KafkaMessage{{Value: []byte("a")}, {Value: []byte("b")}})
	assert.NoError(t, err)
	assert.Equal(t, []KafkaMessage{{Value: []byte("a")}, {Value: []byte("b")}}, got)

	calls := 0
	err = AsBatchHandler(func(context.Context, KafkaMessage) error {
		calls++
		return errors.New("boom")
	})(context.Background(), []KafkaMessage{{Value: []byte("a")}, {Value: []byte("b")}})
	assert.EqualError(t, err, "boom")
	assert.Equal(t, 1, calls)
}

func TestHandleMessageBatchesBlocksForFirstMessage(t *testing.T) {
	ch := make(chan kafka.Message)
	fetch := func(ctx context.Context) (kafka.Message, error) {
		select {
		case <-ctx.Done():
			return kafka.Message{}, ctx.Err()
		case m := <-ch:
			return m, nil
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	var got []KafkaMessage
	go func() {
		handleMessageBatches(ctx, "topic", fetch, func(context.Context, ...kafka.Message) error {
			return nil
		}, 1, time.Hour, func(_ context.Context, msgs []KafkaMessage) error {
			got = msgs
			cancel()
			return nil
		})
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("returned before the first message")
	case <-time.After(30 * time.Millisecond):
	}

	ch <- kafka.Message{Offset: 1, Value: []byte("one")}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("did not return after the first message")
	}
	assert.Equal(t, []KafkaMessage{{Value: []byte("one")}}, got)
}

func TestHandleMessageBatchesMultiMessageUsesMessageTrace(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("OTEL_SAMPLING_RATE", "1.0")
	t.Setenv("OTEL_BSP_EXPORT_TIMEOUT", "1")
	t.Setenv("OTEL_BSP_SCHEDULE_DELAY", "1")
	require.NoError(t, telemetry.Init())
	t.Cleanup(func() { _ = telemetry.Shutdown(context.Background()) })

	traceID, err := trace.TraceIDFromHex("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("bbbbbbbbbbbbbbbb")
	require.NoError(t, err)
	remote := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	headers := telemetry.Inject(trace.ContextWithRemoteSpanContext(context.Background(), remote), nil)
	queued := []kafka.Message{
		{Offset: 1, Value: []byte("a"), Headers: headers},
		{Offset: 2, Value: []byte("b"), Headers: headers},
	}
	calls := 0
	fetch := func(ctx context.Context) (kafka.Message, error) {
		calls++
		if calls > len(queued) {
			<-ctx.Done()
			return kafka.Message{}, ctx.Err()
		}
		return queued[calls-1], nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var got trace.TraceID
	var valid bool
	handleMessageBatches(ctx, "topic", fetch, func(context.Context, ...kafka.Message) error {
		cancel()
		return nil
	}, 2, time.Hour, func(handlerCtx context.Context, _ []KafkaMessage) error {
		sc := trace.SpanFromContext(handlerCtx).SpanContext()
		valid = sc.IsValid()
		got = sc.TraceID()
		return nil
	})
	assert.True(t, valid)
	assert.Equal(t, traceID, got)
}

func TestHandleMessageBatchesStopsAtN(t *testing.T) {
	queued := []kafka.Message{
		{Offset: 1, Value: []byte("a")},
		{Offset: 2, Value: []byte("b")},
		{Offset: 3, Value: []byte("c")},
	}
	calls := 0
	fetch := func(ctx context.Context) (kafka.Message, error) {
		calls++
		if calls > len(queued) {
			<-ctx.Done()
			return kafka.Message{}, ctx.Err()
		}
		return queued[calls-1], nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var got []KafkaMessage
	handleMessageBatches(ctx, "topic", fetch, func(_ context.Context, _ ...kafka.Message) error {
		cancel()
		return nil
	}, 3, time.Hour, func(_ context.Context, msgs []KafkaMessage) error {
		assert.Equal(t, 3, calls)
		got = append([]KafkaMessage(nil), msgs...)
		return nil
	})
	assert.Equal(t, []KafkaMessage{
		{Value: []byte("a")},
		{Value: []byte("b")},
		{Value: []byte("c")},
	}, got)
}

func TestHandleMessageBatchesReturnsPartialBatchOnTimeout(t *testing.T) {
	var calls uint8 = 0
	fetch := func(ctx context.Context) (kafka.Message, error) {
		calls++
		if calls <= 2 {
			return kafka.Message{Offset: int64(calls), Value: []byte{calls}}, nil
		}
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var got []KafkaMessage
	var committed []kafka.Message
	handleMessageBatches(ctx, "topic", fetch, func(_ context.Context, msgs ...kafka.Message) error {
		committed = append(committed, msgs...)
		cancel()
		return nil
	}, 10, 40*time.Millisecond, func(_ context.Context, msgs []KafkaMessage) error {
		got = append([]KafkaMessage(nil), msgs...)
		return nil
	})
	assert.Equal(t, []KafkaMessage{{Value: []byte{1}}, {Value: []byte{2}}}, got)
	assert.Equal(t, []kafka.Message{
		{Offset: 1, Value: []byte{1}},
		{Offset: 2, Value: []byte{2}},
	}, committed)
}

func TestHandleMessageBatchesCancelBeforeFirstMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fetch := func(ctx context.Context) (kafka.Message, error) {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	handled := false
	committed := false
	handleMessageBatches(ctx, "topic", fetch, func(context.Context, ...kafka.Message) error {
		committed = true
		return nil
	}, 5, time.Second, func(context.Context, []KafkaMessage) error {
		handled = true
		return nil
	})
	assert.False(t, handled)
	assert.False(t, committed)
}

func TestHandleMessageBatchesCancelWhileWaitingDropsBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	fetch := func(callCtx context.Context) (kafka.Message, error) {
		calls++
		if calls == 1 {
			return kafka.Message{Offset: 7, Value: []byte("x")}, nil
		}
		cancel()
		<-callCtx.Done()
		return kafka.Message{}, callCtx.Err()
	}
	handled := false
	committed := false
	handleMessageBatches(ctx, "topic", fetch, func(context.Context, ...kafka.Message) error {
		committed = true
		return nil
	}, 5, time.Hour, func(context.Context, []KafkaMessage) error {
		handled = true
		return nil
	})
	assert.False(t, handled)
	assert.False(t, committed)
}

func TestHandleMessageBatchesHandlerErrorSkipsCommit(t *testing.T) {
	fetch := func(context.Context) (kafka.Message, error) {
		return kafka.Message{Value: []byte("a")}, nil
	}
	committed := false
	assert.Panics(t, func() {
		handleMessageBatches(context.Background(), "topic", fetch, func(context.Context, ...kafka.Message) error {
			committed = true
			return nil
		}, 1, time.Second, func(context.Context, []KafkaMessage) error {
			return errors.New("boom")
		})
	})
	assert.False(t, committed)
}
