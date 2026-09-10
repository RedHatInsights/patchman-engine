package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPFilterExcludesHealthAndMetrics(t *testing.T) {
	assert.False(t, TraceHTTPPath("/healthz"))
	assert.False(t, TraceHTTPPath("/livez"))
	assert.False(t, TraceHTTPPath("/readyz"))
	assert.False(t, TraceHTTPPath("/liveness"))
	assert.False(t, TraceHTTPPath("/readiness"))
	assert.False(t, TraceHTTPPath("/metrics"))
	assert.True(t, TraceHTTPPath("/api/patch/v3/advisories"))
}
