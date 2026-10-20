package tasks

import (
	"app/base"
	"app/base/telemetry"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitAndExitFlushesTelemetry(t *testing.T) {
	if os.Getenv("PATCHMAN_TEST_WAIT_AND_EXIT") == "1" {
		require.NoError(t, telemetry.Init())
		_, span := telemetry.Tracer().Start(base.Context, "queued-before-task-exit")
		span.End()
		base.CancelContext()
		WaitAndExit()
		t.Fatal("WaitAndExit returned")
	}

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
	testBinary, err := os.Executable()
	require.NoError(t, err)
	// #nosec G204 -- rerun this test binary with a fixed test name.
	cmd := exec.CommandContext(t.Context(), testBinary, "-test.run=^TestWaitAndExitFlushesTelemetry$")
	cmd.Env = append(os.Environ(), "PATCHMAN_TEST_WAIT_AND_EXIT=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	assert.EqualValues(t, 1, exports.Load(), "forced task exit must flush the queued span")
}
