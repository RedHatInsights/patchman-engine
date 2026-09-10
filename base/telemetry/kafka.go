package telemetry

import (
	"context"
	"strings"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type headerCarrier []kafka.Header

func (c headerCarrier) Get(key string) string {
	for _, h := range c {
		if strings.EqualFold(h.Key, key) {
			return string(h.Value)
		}
	}
	return ""
}

func (c *headerCarrier) Set(key, value string) {
	for i, h := range *c {
		if strings.EqualFold(h.Key, key) {
			(*c)[i] = kafka.Header{Key: h.Key, Value: []byte(value)}
			return
		}
	}
	*c = append(*c, kafka.Header{Key: key, Value: []byte(value)})
}

func (c headerCarrier) Keys() []string {
	keys := make([]string, len(c))
	for i, h := range c {
		keys[i] = h.Key
	}
	return keys
}

func Extract(ctx context.Context, headers []kafka.Header) context.Context {
	c := headerCarrier(headers)
	return otel.GetTextMapPropagator().Extract(ctx, &c)
}

func Inject(ctx context.Context, headers []kafka.Header) []kafka.Header {
	c := headerCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, &c)
	return []kafka.Header(c)
}

func EncodeTraceparent(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	headers := Inject(ctx, nil)
	for _, h := range headers {
		if strings.EqualFold(h.Key, "traceparent") {
			return string(h.Value)
		}
	}
	return ""
}

func ContextFromTraceparent(traceparent string) context.Context {
	if traceparent == "" {
		return context.Background()
	}
	return Extract(context.Background(), []kafka.Header{
		{Key: "traceparent", Value: []byte(traceparent)},
	})
}

func LinksFromTraceparents(tps []string) []trace.Link {
	var links []trace.Link
	for _, tp := range tps {
		if tp == "" {
			continue
		}
		orig := ContextFromTraceparent(tp)
		if sc := trace.SpanContextFromContext(orig); sc.IsValid() {
			links = append(links, trace.Link{SpanContext: sc})
		}
	}
	return links
}

func ConsumerContext(ctx context.Context, topic string, headers []kafka.Header) (context.Context, trace.Span) {
	if !Enabled() || !MQEnabled() {
		return ctx, trace.SpanFromContext(ctx)
	}
	ctx = Extract(ctx, headers)
	return Tracer().Start(ctx, "process "+topic,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.operation", "process"),
		),
	)
}

func ProducerContext(ctx context.Context, topic string, links []trace.Link) (context.Context, trace.Span) {
	if !Enabled() || !MQEnabled() {
		return ctx, trace.SpanFromContext(ctx)
	}
	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.operation", "send"),
		),
	}
	if len(links) > 0 {
		opts = append(opts, trace.WithLinks(links...))
	}
	return Tracer().Start(ctx, "send "+topic, opts...)
}

func ItemContext(parent context.Context, name, traceparent string) (context.Context, trace.Span) {
	if !Enabled() {
		return parent, trace.SpanFromContext(parent)
	}
	opts := []trace.SpanStartOption{trace.WithSpanKind(trace.SpanKindInternal)}
	if orig := ContextFromTraceparent(traceparent); trace.SpanContextFromContext(orig).IsValid() {
		opts = append(opts, trace.WithLinks(trace.LinkFromContext(orig)))
	}
	return Tracer().Start(parent, name, opts...)
}

func SetRHAttributes(span trace.Span, orgID, requestID string) {
	if orgID != "" {
		span.SetAttributes(attribute.String("rh.org_id", orgID))
	}
	if requestID != "" {
		span.SetAttributes(attribute.String("rh.request_id", requestID))
	}
}

func End(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}
