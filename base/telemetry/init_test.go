package telemetry

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestInitReusesProvider(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "true")
	exp := tracetest.NewInMemoryExporter()
	require.NoError(t, initWithExporter(exp))
	initial := tracerProvider
	t.Cleanup(func() { require.NoError(t, initial.Shutdown(context.Background())) })
	t.Cleanup(func() { require.NoError(t, Shutdown(context.Background())) })
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, Init())
			assert.True(t, Enabled())
		}()
	}
	wg.Wait()
	assert.Same(t, initial, tracerProvider)
	assert.Same(t, initial, otel.GetTracerProvider())
	require.NoError(t, Shutdown(context.Background()))
	_, span := initial.Tracer("test").Start(t.Context(), "after-shutdown")
	assert.False(t, span.IsRecording(), "the original provider must be shut down")
	span.End()
}

func TestShutdownConcurrentWithEnabledChecks(t *testing.T) {
	require.NoError(t, initWithExporter(tracetest.NewInMemoryExporter()))
	t.Cleanup(func() { require.NoError(t, Shutdown(context.Background())) })
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = Enabled()
					_ = SQLEnabled()
				}
			}
		}()
	}
	err := Shutdown(context.Background())
	close(stop)
	wg.Wait()
	require.NoError(t, err)
	assert.False(t, Enabled())
}

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
