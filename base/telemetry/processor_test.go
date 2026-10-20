package telemetry

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRHAttributeProcessorConcurrentChildren(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithSpanProcessor(&RHAttributeSpanProcessor{}),
	)
	t.Cleanup(func() { require.NoError(t, tp.Shutdown(context.Background())) })
	tracer := tp.Tracer("test")
	ctx, parent := tracer.Start(t.Context(), "parent")
	parent.SetAttributes(attribute.String("rh.org_id", "org-1"), attribute.String("rh.request_id", "req-1"))
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, child := tracer.Start(ctx, "child")
			child.End()
		}()
	}
	wg.Wait()
	parent.End()
	spans := exp.GetSpans()
	require.Len(t, spans, 33)
	for _, span := range spans {
		assert.Equal(t, "patchman-engine", attr(span, "rh.service"))
		assert.Equal(t, "org-1", attr(span, "rh.org_id"))
		assert.Equal(t, "req-1", attr(span, "rh.request_id"))
	}
}

func TestRHAttributeProcessorSetsServiceAndCopiesParent(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(&RHAttributeSpanProcessor{}),
	)
	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")
	parent.SetAttributes(
		attribute.String("rh.org_id", "org-1"),
		attribute.String("rh.request_id", "req-1"),
	)
	_, child := tp.Tracer("test").Start(ctx, "child")
	child.End()
	parent.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exp.GetSpans()
	require.Len(t, spans, 2)
	var parentStub, childStub tracetest.SpanStub
	for _, s := range spans {
		switch s.Name {
		case "parent":
			parentStub = s
		case "child":
			childStub = s
		}
	}
	assert.Equal(t, "patchman-engine", attr(parentStub, "rh.service"))
	assert.Equal(t, "patchman-engine", attr(childStub, "rh.service"))
	assert.Equal(t, "org-1", attr(childStub, "rh.org_id"))
	assert.Equal(t, "req-1", attr(childStub, "rh.request_id"))
}

func attr(s tracetest.SpanStub, key string) string {
	for _, a := range s.Attributes {
		if string(a.Key) == key {
			return a.Value.AsString()
		}
	}
	return ""
}
