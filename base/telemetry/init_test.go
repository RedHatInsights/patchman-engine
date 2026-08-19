package telemetry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestSQLEnabledAfterInit(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "true")

	exp := tracetest.NewInMemoryExporter()
	require.NoError(t, initWithExporter(exp))
	defer func() {
		_ = Shutdown(context.Background())
	}()

	assert.True(t, SQLEnabled())
}

func TestInitDisabledByDefault(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "")
	require.NoError(t, Init())
	defer func() {
		_ = Shutdown(context.Background())
	}()

	assert.False(t, Enabled())
	_, span := Tracer().Start(context.Background(), "noop")
	assert.False(t, span.SpanContext().IsValid() && span.IsRecording())
	span.End()
}

func TestInitPreservesSpanEventLimits(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "true")

	exp := tracetest.NewInMemoryExporter()
	require.NoError(t, initWithExporter(exp))
	defer func() {
		_ = Shutdown(context.Background())
	}()

	_, span := Tracer().Start(context.Background(), "event-limit-test")
	span.RecordError(errors.New("test error"))
	span.End()
	require.NoError(t, tracerProvider.ForceFlush(context.Background()))

	spans := exp.GetSpans()
	require.Len(t, spans, 1)
	assert.Greater(t, len(spans[0].Events), 0)
	assert.Equal(t, 0, spans[0].DroppedEvents)
}

func TestInitSetsResourceAndSampler(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("OTEL_SERVICE_NAME", "patchman-listener")
	t.Setenv("IMAGE_TAG", "test-tag")
	t.Setenv("OTEL_SAMPLING_RATE", "0.5")

	exp := tracetest.NewInMemoryExporter()
	require.NoError(t, initWithExporter(exp))
	defer func() {
		_ = Shutdown(context.Background())
	}()

	res := providerResource
	require.NotNil(t, res)
	assert.Contains(t, fmtAttrs(res.Attributes()), "service.name=patchman-listener")
	assert.Contains(t, fmtAttrs(res.Attributes()), "service.version=test-tag")

	sampler := providerSampler
	sampledParent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    mustTraceID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		SpanID:     mustSpanID("bbbbbbbbbbbbbbbb"),
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	unsampledParent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: mustTraceID("cccccccccccccccccccccccccccccccc"),
		SpanID:  mustSpanID("dddddddddddddddd"),
		Remote:  true,
	})
	assert.Equal(t, sdktrace.RecordAndSample, sampler.ShouldSample(sdktrace.SamplingParameters{
		ParentContext: trace.ContextWithRemoteSpanContext(context.Background(), sampledParent),
	}).Decision)
	assert.Equal(t, sdktrace.Drop, sampler.ShouldSample(sdktrace.SamplingParameters{
		ParentContext: trace.ContextWithRemoteSpanContext(context.Background(), unsampledParent),
	}).Decision)
}
