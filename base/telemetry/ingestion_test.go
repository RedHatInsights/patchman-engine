package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestIngestionCompletionContract(t *testing.T) {
	previous := enabled.Load()
	enabled.Store(true)
	t.Cleanup(func() { enabled.Store(previous) })
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
	for _, test := range []struct {
		name, orgID, requestID, hostID, outcome string
		disabled                                bool
	}{
		{name: "committed", orgID: "org", requestID: "request", hostID: "host", outcome: "processed"},
		{name: "no dispatch", orgID: "org", requestID: "request", hostID: "host", outcome: "unchanged"},
		{name: "missing request", orgID: "org", hostID: "host"},
		{name: "missing org", requestID: "request", hostID: "host"},
		{name: "missing host", orgID: "org", requestID: "request"},
		{name: "MQ disabled", orgID: "org", requestID: "request", hostID: "host", disabled: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("OTEL_MQ_ENABLED", "true")
			if test.disabled {
				t.Setenv("OTEL_MQ_ENABLED", "false")
			}
			exporter.Reset()
			ctx, span := provider.Tracer("test").Start(t.Context(), test.name)
			before := time.Now()
			RecordIngestionCompletion(ctx, test.orgID, test.requestID, test.hostID, test.outcome)
			after := time.Now()
			span.End()
			events := exporter.GetSpans()[0].Events
			if test.outcome == "" {
				assert.Empty(t, events)
				return
			}
			require.Len(t, events, 1)
			assert.Equal(t, "rh.ingestion.completed", events[0].Name)
			assert.False(t, events[0].Time.Before(before))
			assert.False(t, events[0].Time.After(after))
			attrs := map[string]string{}
			for _, a := range events[0].Attributes {
				attrs[string(a.Key)] = a.Value.AsString()
			}
			assert.Equal(t, map[string]string{
				"rh.ingestion.application": "patch", "rh.ingestion.milestone": "processing",
				"rh.ingestion.outcome": test.outcome, "rh.org_id": "org",
				"rh.request_id": "request", "rh.host_id": "host",
			}, attrs)
		})
	}
}
