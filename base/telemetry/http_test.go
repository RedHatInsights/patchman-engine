package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
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

func TestHTTPServerSpanNames(t *testing.T) {
	t.Setenv("OTEL_HTTP_INBOUND_ENABLED", "true")
	t.Setenv("OTEL_SAMPLING_RATE", "1")
	for _, tc := range []struct {
		name       string
		method     string
		path       string
		route      string
		spanName   string
		statusCode int
	}{
		{"list", http.MethodGet, "/api/patch/v3/systems?limit=5", "/api/patch/v3/systems",
			"GET /api/patch/v3/systems", http.StatusOK},
		{"first system", http.MethodGet, "/api/patch/v3/systems/first", "/api/patch/v3/systems/:inventory_id",
			"GET /api/patch/v3/systems/:inventory_id", http.StatusOK},
		{"second system", http.MethodGet, "/api/patch/v3/systems/second", "/api/patch/v3/systems/:inventory_id",
			"GET /api/patch/v3/systems/:inventory_id", http.StatusOK},
		{"post", http.MethodPost, "/api/patch/v3/systems", "/api/patch/v3/systems",
			"POST /api/patch/v3/systems", http.StatusOK},
		{"invalid system", http.MethodGet, "/api/patch/v3/systems/invalid", "/api/patch/v3/systems/:inventory_id",
			"GET /api/patch/v3/systems/:inventory_id", http.StatusBadRequest},
		{"missing system", http.MethodGet, "/api/patch/v3/systems/missing", "/api/patch/v3/systems/:inventory_id",
			"GET /api/patch/v3/systems/:inventory_id", http.StatusNotFound},
		{"unmatched path", http.MethodGet, "/api/patch/v3/unknown", "", "GET", http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exporter := tracetest.NewInMemoryExporter()
			require.NoError(t, initWithExporter(exporter))
			t.Cleanup(func() { require.NoError(t, Shutdown(context.Background())) })
			app := gin.New()
			app.Use(RHHTTPAttributes())
			if tc.route != "" {
				app.Handle(tc.method, tc.route, func(c *gin.Context) { c.AbortWithStatus(tc.statusCode) })
			}
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("x-rh-insights-request-id", "test-request")
			response := httptest.NewRecorder()
			InstrumentHandler(app).ServeHTTP(response, req)

			assert.Equal(t, tc.statusCode, response.Code)
			spans := exporter.GetSpans()
			require.Len(t, spans, 1, "route naming must reuse the HTTP server span")
			assert.Equal(t, tc.spanName, spans[0].Name)
			assert.Equal(t, trace.SpanKindServer, spans[0].SpanKind)
			assert.False(t, spans[0].Parent.IsValid())
			assert.Equal(t, tc.route, attr(spans[0], "http.route"))
			assert.Equal(t, "test-request", attr(spans[0], "rh.request_id"))
		})
	}
}
