package telemetry

import (
	"app/base/utils"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

var (
	enabled          bool
	tracerProvider   *sdktrace.TracerProvider
	providerResource *resource.Resource
	providerSampler  sdktrace.Sampler
)

func Init() error {
	if !otelEnabled() {
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		enabled = false
		utils.LogInfo("OpenTelemetry is disabled")
		return nil
	}

	exp, err := otlptracehttp.New(context.Background(), otlpCompressionOptions()...)
	if err != nil {
		return err
	}

	return initTracerProvider(exp, true)
}

func initWithExporter(exp sdktrace.SpanExporter) error {
	return initTracerProvider(exp, false)
}

func initTracerProvider(exp sdktrace.SpanExporter, useBatch bool) error {
	res, err := newResource()
	if err != nil {
		return err
	}

	providerResource = res
	providerSampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplingRate()))

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(providerSampler),
		sdktrace.WithSpanProcessor(&RHAttributeSpanProcessor{}),
		sdktrace.WithRawSpanLimits(sdktrace.SpanLimits{
			AttributeCountLimit:       utils.GetIntEnvOrDefault("OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT", 64),
			AttributeValueLengthLimit: utils.GetIntEnvOrDefault("OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT", 1024),
		}),
	}

	if useBatch {
		opts = append(opts, sdktrace.WithBatcher(exp,
			sdktrace.WithMaxQueueSize(utils.GetIntEnvOrDefault("OTEL_BSP_MAX_QUEUE_SIZE", 8192)),
			sdktrace.WithMaxExportBatchSize(utils.GetIntEnvOrDefault("OTEL_BSP_MAX_EXPORT_BATCH_SIZE", 256)),
			sdktrace.WithBatchTimeout(durationFromMsEnv("OTEL_BSP_SCHEDULE_DELAY", 2*time.Second)),
			sdktrace.WithExportTimeout(durationFromMsEnv("OTEL_BSP_EXPORT_TIMEOUT", 10*time.Second)),
		))
	} else {
		opts = append(opts, sdktrace.WithSyncer(exp))
	}

	tracerProvider = sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	enabled = true
	return nil
}

func Shutdown(ctx context.Context) error {
	if tracerProvider == nil {
		return nil
	}
	err := tracerProvider.Shutdown(ctx)
	tracerProvider = nil
	providerResource = nil
	providerSampler = nil
	enabled = false
	return err
}

func Tracer() trace.Tracer {
	return otel.Tracer("app/base/telemetry")
}

func newResource() (*resource.Resource, error) {
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = RHServiceName
	}
	version := os.Getenv("IMAGE_TAG")
	if version == "" {
		version = "unknown"
	}
	deployEnv := os.Getenv("NAMESPACE")
	if deployEnv == "" {
		deployEnv = "development"
	}
	return resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(deployEnv),
		),
	)
}

func otlpCompressionOptions() []otlptracehttp.Option {
	comp := strings.ToLower(os.Getenv("OTEL_EXPORTER_OTLP_COMPRESSION"))
	if comp == "" || comp == "none" {
		return nil
	}
	return []otlptracehttp.Option{otlptracehttp.WithCompression(otlptracehttp.GzipCompression)}
}

func durationFromMsEnv(key string, def time.Duration) time.Duration {
	ms := utils.GetIntEnvOrDefault(key, int(def/time.Millisecond))
	return time.Duration(ms) * time.Millisecond
}

func fmtAttrs(attrs []attribute.KeyValue) []string {
	result := make([]string, 0, len(attrs))
	for _, a := range attrs {
		result = append(result, fmt.Sprintf("%s=%s", a.Key, a.Value.AsString()))
	}
	return result
}

func mustTraceID(s string) trace.TraceID {
	id, err := trace.TraceIDFromHex(s)
	if err != nil {
		panic(err)
	}
	return id
}

func mustSpanID(s string) trace.SpanID {
	id, err := trace.SpanIDFromHex(s)
	if err != nil {
		panic(err)
	}
	return id
}
