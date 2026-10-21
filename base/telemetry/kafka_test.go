package telemetry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
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

func TestDisabledProducerDoesNotEndParent(t *testing.T) {
	exp := setupKafkaTest(t)
	t.Setenv("OTEL_MQ_ENABLED", "false")
	ctx, parent := Tracer().Start(t.Context(), "evaluate upload")
	producerCtx, producer := ProducerContext(ctx, "platform.inventory.host-apps", nil)

	assert.Same(t, ctx, producerCtx)
	assert.Equal(t, parent.SpanContext(), trace.SpanContextFromContext(producerCtx))
	End(producer, errors.New("Kafka unavailable"))
	assert.True(t, parent.IsRecording(), "ending a disabled producer must not end its parent")
	assert.Empty(t, exp.GetSpans(), "the parent must remain open after publishing")
	parent.End()

	spans := flushSpans(t, exp)
	require.Len(t, spans, 1)
	assert.Equal(t, "evaluate upload", spans[0].Name)
	assert.Equal(t, codes.Unset, spans[0].Status.Code)
	assert.Empty(t, spans[0].Events, "a disabled producer must not record errors on the parent")
}

func TestItemContextParentsOriginalTrace(t *testing.T) {
	exp := setupKafkaTest(t)

	origCtx, orig := Tracer().Start(context.Background(), "original-item")
	originalTraceparent := EncodeTraceparent(origCtx)
	orig.End()

	producerCtx, producer := ProducerContext(ContextFromTraceparent(originalTraceparent), "patchman.evaluator.upload", nil)
	producerHeaders := Inject(producerCtx, nil)
	producer.End()

	bulkCtx, bulk := ConsumerContext(context.Background(), "patchman.evaluator.upload", producerHeaders)
	_, item := ItemContext(bulkCtx, "evaluate upload", originalTraceparent)
	item.End()
	bulk.End()

	spans := flushSpans(t, exp)
	var bulkStub, itemStub, producerStub tracetest.SpanStub
	for _, s := range spans {
		switch s.Name {
		case "process patchman.evaluator.upload":
			bulkStub = s
		case "evaluate upload":
			itemStub = s
		case "send patchman.evaluator.upload":
			producerStub = s
		}
	}
	require.NotEmpty(t, bulkStub.Name)
	require.NotEmpty(t, itemStub.Name)
	require.NotEmpty(t, producerStub.Name)

	origSC := trace.SpanContextFromContext(ContextFromTraceparent(originalTraceparent))
	assert.Equal(t, origSC.TraceID(), itemStub.SpanContext.TraceID())
	assert.Equal(t, origSC.SpanID(), itemStub.Parent.SpanID())
	assert.Empty(t, itemStub.Links)

	assert.Equal(t, origSC.TraceID(), producerStub.SpanContext.TraceID())
	assert.Equal(t, origSC.SpanID(), producerStub.Parent.SpanID())
	assert.Equal(t, producerStub.SpanContext.SpanID(), bulkStub.Parent.SpanID())
}

func TestItemContextPreservesCallerContext(t *testing.T) {
	setupKafkaTest(t)
	originalCtx, original := Tracer().Start(t.Context(), "original")
	tp := EncodeTraceparent(originalCtx)
	original.End()
	for _, parentTrace := range []string{tp, "", "invalid"} {
		t.Run(parentTrace, func(t *testing.T) {
			type contextKey struct{}
			deadline := time.Now().Add(time.Minute)
			caller, cancel := context.WithDeadline(context.WithValue(t.Context(), contextKey{}, "request-state"), deadline)
			defer cancel()
			ctx, span := ItemContext(caller, "evaluate upload", parentTrace)
			defer span.End()

			assert.Equal(t, "request-state", ctx.Value(contextKey{}))
			actualDeadline, ok := ctx.Deadline()
			assert.True(t, ok)
			assert.Equal(t, deadline, actualDeadline)
			if parentTrace == tp {
				assert.Equal(t, original.SpanContext().TraceID(), span.SpanContext().TraceID())
			}
			cancel()
			assert.ErrorIs(t, ctx.Err(), context.Canceled)
		})
	}
}

func TestParentContextFromTraceparents(t *testing.T) {
	setupKafkaTest(t)

	ctx0, span0 := Tracer().Start(context.Background(), "item-0")
	tp0 := EncodeTraceparent(ctx0)
	span0.End()

	ctx1, span1 := Tracer().Start(context.Background(), "item-1")
	tp1 := EncodeTraceparent(ctx1)
	span1.End()

	got := ParentContextFromTraceparents([]string{"", tp0, tp1})
	sc := trace.SpanContextFromContext(got)
	origSC := trace.SpanContextFromContext(ContextFromTraceparent(tp0))
	assert.Equal(t, origSC.TraceID(), sc.TraceID())
	assert.Equal(t, origSC.SpanID(), sc.SpanID())

	assert.Equal(t, context.Background(), ParentContextFromTraceparents(nil))
	assert.Equal(t, context.Background(), ParentContextFromTraceparents([]string{"", ""}))
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
