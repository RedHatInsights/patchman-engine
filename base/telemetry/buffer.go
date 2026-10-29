package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// RecordBufferWait records one host's additional wait before publication. Time
// already covered by its local processing span is excluded. A still-running
// processing span (including a synchronous flush) therefore gets a zero-duration
// observation, keeping span-metric counts representative of all buffered hosts.
func RecordBufferWait(parent trace.Span, topic string, enqueuedAt, publishAt time.Time) {
	if !MQEnabled() {
		return
	}
	processing, ok := parent.(sdktrace.ReadOnlySpan)
	if !ok {
		// Remote and unsampled spans cannot supply a local processing end time.
		return
	}
	start := enqueuedAt
	processedAt := processing.EndTime()
	if processedAt.IsZero() || processedAt.After(publishAt) {
		start = publishAt
	} else if processedAt.After(start) {
		start = processedAt
	}

	ctx := trace.ContextWithSpan(context.Background(), parent)
	_, span := Tracer().Start(ctx, "buffer wait "+topic,
		trace.WithTimestamp(start),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attribute.String("messaging.destination.name", topic)),
	)
	span.End(trace.WithTimestamp(publishAt))
}
