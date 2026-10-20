package main

import (
	"app/base/telemetry"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainFlushesTelemetryOnNormalReturn(t *testing.T) {
	var exports atomic.Int32
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	_, span := telemetry.Tracer().Start(t.Context(), "queued-before-return")
	span.End()
	previousArgs := os.Args
	os.Args = []string{"patchman", "job", "no-job-selected"}
	t.Cleanup(func() { os.Args = previousArgs })
	main()
	assert.EqualValues(t, 1, exports.Load(), "normal return must flush the queued span")
	assert.False(t, telemetry.Enabled())
}
