package listener

import (
	"app/base/mqueue"
	"app/base/telemetry"
	"app/base/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type bufferTestWriter func(context.Context, ...mqueue.KafkaMessage) error

func (w bufferTestWriter) WriteMessages(ctx context.Context, messages ...mqueue.KafkaMessage) error {
	return w(ctx, messages...)
}

func setupBufferTest(t *testing.T, size int) *tracetest.SpanRecorder {
	t.Helper()
	previousProvider, previousPropagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	previousSize, previousTimeout := eventBufferSize, uploadEvalTimeout
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("OTEL_MQ_ENABLED", "true")
	require.NoError(t, telemetry.Init())
	// Use an in-memory provider; no spans go to the production OTLP exporter.
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
		sdktrace.WithSpanProcessor(&telemetry.RHAttributeSpanProcessor{}),
	)
	otel.SetTracerProvider(provider)
	eventBufferSize, uploadEvalTimeout = size, time.Hour
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(context.Background()))
		require.NoError(t, telemetry.Shutdown(context.Background()))
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
		eventBufferSize, uploadEvalTimeout = previousSize, previousTimeout
	})
	return recorder
}

func newBufferTestEvent() *mqueue.PayloadTrackerEvent {
	return &mqueue.PayloadTrackerEvent{
		InventoryID: uuid.New(),
		OrgID:       utils.PtrString("buffer-org"),
		RequestID:   utils.PtrString(uuid.NewString()),
	}
}

func findBufferTestSpans(recorder *tracetest.SpanRecorder, name string) []sdktrace.ReadOnlySpan {
	var spans []sdktrace.ReadOnlySpan
	for _, span := range recorder.Ended() {
		if span.Name() == name {
			spans = append(spans, span)
		}
	}
	return spans
}

func TestEventBufferProducerTopicAndLifetime(t *testing.T) {
	for _, destination := range []string{"upload", "user-evaluation"} {
		for _, failure := range []string{"none", "evaluation", "payload-tracker"} {
			t.Run(destination+"/"+failure, func(t *testing.T) {
				testBufferProducer(t, "patchman.evaluator."+destination, failure)
			})
		}
	}
}

func testBufferProducer(t *testing.T, topic, failure string) {
	t.Helper()
	recorder := setupBufferTest(t, 1)
	ctx, parent := telemetry.Tracer().Start(t.Context(), "processing")
	defer parent.End()
	event := newBufferTestEvent()
	var producer trace.Span
	var evaluationReturnedAt time.Time
	var payloadCalls int
	evaluationWriter := mqueue.Writer(bufferTestWriter(func(ctx context.Context, messages ...mqueue.KafkaMessage) error {
		producer = trace.SpanFromContext(ctx)
		assert.True(t, producer.IsRecording())
		require.Len(t, messages, 1)
		var published mqueue.PlatformEvent
		require.NoError(t, sonic.Unmarshal(messages[0].Value, &published))
		assert.Equal(t, []uuid.UUID{event.InventoryID}, published.SystemIDs)
		assert.Equal(t, []string{telemetry.EncodeTraceparent(trace.ContextWithSpan(t.Context(), parent))},
			published.Traceparents)
		evaluationReturnedAt = time.Now()
		if failure == "evaluation" {
			return errors.New("evaluation publish failed")
		}
		return nil
	}))
	payloadWriter := mqueue.Writer(bufferTestWriter(func(context.Context, ...mqueue.KafkaMessage) error {
		payloadCalls++
		require.NotNil(t, producer)
		assert.False(t, producer.IsRecording(), "producer must end before payload tracking starts")
		if failure == "payload-tracker" {
			return errors.New("payload tracking failed")
		}
		return nil
	}))
	var buffer eventBuffer
	buffer.initEventBuffer(&evaluationWriter, &payloadWriter, topic)
	t.Cleanup(func() { buffer.flushTimer.Stop() })
	buffer.bufferEvalEvents(event.InventoryID, 1, event, ctx)
	assert.Equal(t, 1, payloadCalls)
	spans := findBufferTestSpans(recorder, "send "+topic)
	require.Len(t, spans, 1)
	assert.Equal(t, trace.SpanKindProducer, spans[0].SpanKind())
	assert.Contains(t, spans[0].Attributes(), attribute.String("messaging.destination.name", topic))
	assert.False(t, spans[0].EndTime().Before(evaluationReturnedAt))
	wantStatus := codes.Unset
	if failure == "evaluation" {
		wantStatus = codes.Error
	}
	assert.Equal(t, wantStatus, spans[0].Status().Code)
	buffer.flushEvalEvents()
	assert.Len(t, findBufferTestSpans(recorder, "send "+topic), 1, "empty flush must not emit a producer")
}

func TestEventBufferFullFlushWaitPerHost(t *testing.T) {
	recorder := setupBufferTest(t, 2)
	const topic = "full-buffer-evaluation"
	writer := mqueue.Writer(&mqueue.MockKafkaWriter{})
	var buffer eventBuffer
	buffer.initEventBuffer(&writer, &writer, topic)
	t.Cleanup(func() { buffer.flushTimer.Stop() })
	ctx1, parent1 := telemetry.Tracer().Start(t.Context(), "first host")
	event1 := newBufferTestEvent()
	telemetry.SetRHAttributes(parent1, *event1.OrgID, *event1.RequestID)
	buffer.bufferEvalEvents(event1.InventoryID, 1, event1, ctx1)
	parent1.End()
	ctx2, parent2 := telemetry.Tracer().Start(t.Context(), "second host")
	event2 := newBufferTestEvent()
	telemetry.SetRHAttributes(parent2, *event2.OrgID, *event2.RequestID)
	buffer.bufferEvalEvents(event2.InventoryID, 1, event2, ctx2)
	assert.True(t, parent2.IsRecording())
	parent2.End()

	waits := findBufferTestSpans(recorder, "buffer wait "+topic)
	require.Len(t, waits, 2)
	assert.Equal(t, parent1.SpanContext().TraceID(), waits[0].SpanContext().TraceID())
	assert.Equal(t, parent1.SpanContext().SpanID(), waits[0].Parent().SpanID())
	assert.Equal(t, parent1.(sdktrace.ReadOnlySpan).EndTime(), waits[0].StartTime())
	assert.True(t, waits[0].EndTime().After(waits[0].StartTime()))
	assert.Equal(t, parent2.SpanContext().TraceID(), waits[1].SpanContext().TraceID())
	assert.Equal(t, parent2.SpanContext().SpanID(), waits[1].Parent().SpanID())
	assert.Equal(t, waits[1].StartTime(), waits[1].EndTime(), "synchronous host's wait is already in processing")
	for i, event := range []*mqueue.PayloadTrackerEvent{event1, event2} {
		assert.Contains(t, waits[i].Attributes(), attribute.String("rh.request_id", *event.RequestID))
	}
	producer := findBufferTestSpans(recorder, "send "+topic)
	require.Len(t, producer, 1)
	assert.False(t, waits[0].EndTime().After(producer[0].StartTime()), "buffer wait excludes publishing")
	buffer.flushEvalEvents()
	assert.Len(t, findBufferTestSpans(recorder, "buffer wait "+topic), 2)
}

func TestEventBufferTimerWait(t *testing.T) {
	recorder := setupBufferTest(t, 2)
	const topic = "timer-evaluation"
	evaluationWriter := mqueue.Writer(&mqueue.MockKafkaWriter{})
	done := make(chan struct{})
	payloadWriter := mqueue.Writer(bufferTestWriter(func(context.Context, ...mqueue.KafkaMessage) error {
		close(done)
		return nil
	}))
	var buffer eventBuffer
	buffer.initEventBuffer(&evaluationWriter, &payloadWriter, topic)
	t.Cleanup(func() { buffer.flushTimer.Stop() })
	ctx, parent := telemetry.Tracer().Start(t.Context(), "timer host")
	event := newBufferTestEvent()
	buffer.bufferEvalEvents(event.InventoryID, 1, event, ctx)
	parent.End()
	buffer.flushTimer.Reset(time.Millisecond)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("buffer timer did not publish")
	}
	buffer.lock.Lock()
	defer buffer.lock.Unlock()
	waits := findBufferTestSpans(recorder, "buffer wait "+topic)
	require.Len(t, waits, 1)
	assert.Equal(t, parent.(sdktrace.ReadOnlySpan).EndTime(), waits[0].StartTime())
	assert.True(t, waits[0].EndTime().After(waits[0].StartTime()))
	assert.Empty(t, buffer.evalBuffer)
	assert.Empty(t, buffer.traces)
}
