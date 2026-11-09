package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RecordIngestionCompletion records native processing evidence for the external correlator.
// Call only after results commit or a successful terminal no-work decision.
func RecordIngestionCompletion(ctx context.Context, orgID, requestID, hostID, outcome string) {
	span := trace.SpanFromContext(ctx)
	if !MQEnabled() || !span.IsRecording() || orgID == "" || requestID == "" || hostID == "" {
		return
	}
	span.AddEvent("rh.ingestion.completed", trace.WithAttributes(
		attribute.String("rh.ingestion.application", "patch"),
		attribute.String("rh.ingestion.milestone", "processing"),
		attribute.String("rh.ingestion.outcome", outcome),
		attribute.String("rh.org_id", orgID),
		attribute.String("rh.request_id", requestID),
		attribute.String("rh.host_id", hostID),
	))
}
