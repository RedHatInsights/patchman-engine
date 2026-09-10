package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type RHAttributeSpanProcessor struct{}

func (p *RHAttributeSpanProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	s.SetAttributes(attribute.String("rh.service", RHServiceName))

	parentSpan := trace.SpanFromContext(parent)
	if parentSpan == nil {
		return
	}

	readOnly, ok := parentSpan.(sdktrace.ReadOnlySpan)
	if !ok {
		return
	}

	currentKeys := make(map[string]bool)
	for _, a := range s.Attributes() {
		currentKeys[string(a.Key)] = true
	}

	for _, a := range readOnly.Attributes() {
		key := string(a.Key)
		if (key == "rh.org_id" || key == "rh.request_id") && !currentKeys[key] {
			s.SetAttributes(a)
		}
	}
}

func (p *RHAttributeSpanProcessor) OnEnd(sdktrace.ReadOnlySpan) {}

func (p *RHAttributeSpanProcessor) Shutdown(context.Context) error { return nil }

func (p *RHAttributeSpanProcessor) ForceFlush(context.Context) error { return nil }
