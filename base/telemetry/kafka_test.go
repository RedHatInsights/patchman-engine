package telemetry

import (
	"context"
	"strings"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func setupKafkaTest(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("OTEL_SAMPLING_RATE", "1.0")
	exp := tracetest.NewInMemoryExporter()
	require.NoError(t, initWithExporter(exp))
	t.Cleanup(func() { _ = Shutdown(context.Background()) })
	return exp
}

func flushSpans(t *testing.T, exp *tracetest.InMemoryExporter) []tracetest.SpanStub {
	t.Helper()
	if tracerProvider != nil {
		require.NoError(t, tracerProvider.ForceFlush(context.Background()))
	}
	return exp.GetSpans()
}

func TestConsumerContextIsChildNotLink(t *testing.T) {
	exp := setupKafkaTest(t)

	remote := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    mustTraceID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		SpanID:     mustSpanID("bbbbbbbbbbbbbbbb"),
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	headers := Inject(trace.ContextWithRemoteSpanContext(context.Background(), remote), nil)

	_, span := ConsumerContext(context.Background(), "platform.inventory.events", headers)
	span.End()

	spans := flushSpans(t, exp)
	require.Len(t, spans, 1)
	child := spans[0]
	assert.Equal(t, remote.TraceID(), child.SpanContext.TraceID())
	assert.Equal(t, remote.SpanID(), child.Parent.SpanID())
	assert.Empty(t, child.Links)
}

func TestProducerContextLinksOriginalsAndInjectsOwnTraceparent(t *testing.T) {
	exp := setupKafkaTest(t)

	ctx0, item0 := Tracer().Start(context.Background(), "item-0")
	tp0 := EncodeTraceparent(ctx0)
	item0.End()

	ctx1, item1 := Tracer().Start(context.Background(), "item-1")
	tp1 := EncodeTraceparent(ctx1)
	item1.End()

	links := LinksFromTraceparents([]string{tp0, tp1})
	producerCtx, producer := ProducerContext(context.Background(), "patchman.evaluator.upload", links)
	producerHeaders := Inject(producerCtx, nil)
	producer.End()

	spans := flushSpans(t, exp)
	var producerStub tracetest.SpanStub
	for _, s := range spans {
		if s.Name == "send patchman.evaluator.upload" {
			producerStub = s
		}
	}
	require.NotEmpty(t, producerStub.Name)
	assert.False(t, producerStub.Parent.IsValid())
	require.Len(t, producerStub.Links, 2)

	item0SC := trace.SpanContextFromContext(ContextFromTraceparent(tp0))
	item1SC := trace.SpanContextFromContext(ContextFromTraceparent(tp1))
	linkIDs := []trace.SpanID{producerStub.Links[0].SpanContext.SpanID(), producerStub.Links[1].SpanContext.SpanID()}
	assert.Contains(t, linkIDs, item0SC.SpanID())
	assert.Contains(t, linkIDs, item1SC.SpanID())

	extracted := Extract(context.Background(), producerHeaders)
	_, child := Tracer().Start(extracted, "downstream")
	child.End()

	spans = flushSpans(t, exp)
	var childStub tracetest.SpanStub
	for _, s := range spans {
		if s.Name == "downstream" {
			childStub = s
		}
	}
	require.NotEmpty(t, childStub.Name)
	assert.Equal(t, producerStub.SpanContext.TraceID(), childStub.SpanContext.TraceID())
}

func TestItemContextLinksOriginalAndParentsBulk(t *testing.T) {
	exp := setupKafkaTest(t)

	origCtx, orig := Tracer().Start(context.Background(), "original-item")
	originalTraceparent := EncodeTraceparent(origCtx)
	orig.End()

	producerCtx, producer := ProducerContext(context.Background(), "patchman.evaluator.upload", nil)
	producerHeaders := Inject(producerCtx, nil)
	producer.End()

	bulkCtx, bulk := ConsumerContext(context.Background(), "patchman.evaluator.upload", producerHeaders)
	_, item := ItemContext(bulkCtx, "evaluate upload", originalTraceparent)
	item.End()
	bulk.End()

	spans := flushSpans(t, exp)
	var bulkStub, itemStub tracetest.SpanStub
	for _, s := range spans {
		switch s.Name {
		case "process patchman.evaluator.upload":
			bulkStub = s
		case "evaluate upload":
			itemStub = s
		}
	}
	require.NotEmpty(t, bulkStub.Name)
	require.NotEmpty(t, itemStub.Name)
	assert.Equal(t, bulkStub.SpanContext.SpanID(), itemStub.Parent.SpanID())

	origSC := trace.SpanContextFromContext(ContextFromTraceparent(originalTraceparent))
	require.Len(t, itemStub.Links, 1)
	assert.Equal(t, origSC.TraceID(), itemStub.Links[0].SpanContext.TraceID())
	assert.Equal(t, origSC.SpanID(), itemStub.Links[0].SpanContext.SpanID())
}

func TestEncodeRoundTrip(t *testing.T) {
	setupKafkaTest(t)

	ctx, span := Tracer().Start(context.Background(), "x")
	defer span.End()
	tp := EncodeTraceparent(ctx)
	require.Len(t, strings.Split(tp, "-")[1], 32)
	got := trace.SpanContextFromContext(ContextFromTraceparent(tp))
	assert.Equal(t, span.SpanContext().TraceID(), got.TraceID())
	assert.Equal(t, span.SpanContext().SpanID(), got.SpanID())
}

func TestHeaderCarrierCaseInsensitive(t *testing.T) {
	setupKafkaTest(t)

	headers := []kafka.Header{{Key: "TraceParent", Value: []byte("old")}}
	remote := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    mustTraceID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		SpanID:     mustSpanID("bbbbbbbbbbbbbbbb"),
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	out := Inject(trace.ContextWithRemoteSpanContext(context.Background(), remote), headers)
	require.Len(t, out, 1)
	assert.Equal(t, "TraceParent", out[0].Key)
	assert.NotEqual(t, "old", string(out[0].Value))
}
