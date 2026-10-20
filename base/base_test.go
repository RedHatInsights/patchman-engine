package base

import (
	"app/base/telemetry"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShutdownTelemetryWithCanceledApplicationContext(t *testing.T) {
	var exports atomic.Int32
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/traces", r.URL.Path)
		exports.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer collector.Close()
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("OTEL_SAMPLING_RATE", "1")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_COMPRESSION", "none")
	t.Setenv("OTEL_BSP_SCHEDULE_DELAY", "3600000")
	require.NoError(t, telemetry.Init())
	t.Cleanup(func() { require.NoError(t, telemetry.Shutdown(context.Background())) })

	previousContext := Context
	ctx, cancel := context.WithCancel(previousContext)
	Context = ctx
	t.Cleanup(func() { Context = previousContext })
	_, span := telemetry.Tracer().Start(ctx, "queued-before-shutdown")
	span.End()
	cancel()
	ShutdownTelemetry()
	assert.EqualValues(t, 1, exports.Load(), "shutdown must flush the queued span")
	assert.False(t, telemetry.Enabled())
}
