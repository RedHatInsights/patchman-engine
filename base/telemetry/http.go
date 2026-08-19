package telemetry

import (
	"net/http"

	"app/base/utils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	// Avoid utils→telemetry import cycle: RunServer calls InstrumentHTTPHandler.
	utils.InstrumentHTTPHandler = InstrumentHandler
}

// TraceHTTPPath reports whether inbound HTTP tracing should create a span for path.
func TraceHTTPPath(path string) bool {
	switch path {
	case "/healthz", "/livez", "/readyz", "/liveness", "/readiness", "/metrics":
		return false
	}
	if mp := utils.CoreCfg.MetricsPath; mp != "" && path == mp {
		return false
	}
	return true
}

// InstrumentHandler wraps h with otelhttp when inbound HTTP tracing is enabled.
func InstrumentHandler(h http.Handler) http.Handler {
	if !HTTPInboundEnabled() {
		return h
	}
	return otelhttp.NewHandler(h, "http.server",
		otelhttp.WithFilter(func(r *http.Request) bool {
			return TraceHTTPPath(r.URL.Path)
		}),
	)
}

// InstrumentHTTPClient wraps c.Transport with otelhttp when outbound HTTP tracing is enabled.
func InstrumentHTTPClient(c *http.Client) *http.Client {
	if c == nil {
		c = &http.Client{}
	}
	if !HTTPOutboundEnabled() {
		return c
	}
	if _, ok := c.Transport.(*otelhttp.Transport); ok {
		return c
	}
	base := c.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.Transport = otelhttp.NewTransport(base)
	return c
}

// RHHTTPAttributes sets rh.request_id and rh.org_id on the current span from request headers.
func RHHTTPAttributes() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if reqID := c.GetHeader("x-rh-insights-request-id"); reqID != "" {
			span.SetAttributes(attribute.String("rh.request_id", reqID))
		}
		if identity := c.GetHeader("x-rh-identity"); identity != "" {
			if xrhid, err := utils.ParseXRHID(identity); err == nil {
				span.SetAttributes(attribute.String("rh.org_id", xrhid.Identity.OrgID))
			}
		}
		c.Next()
	}
}
