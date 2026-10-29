package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestRecordBufferWaitExcludesProcessingOverlap(t *testing.T) {
	const topic = "evaluation"
	enqueuedAt := time.Now()
	publishAt := enqueuedAt.Add(time.Second)
	cases := []struct {
		name        string
		processedAt time.Time
		wantStart   time.Time
	}{
		{"already processed", enqueuedAt.Add(-time.Millisecond), enqueuedAt},
		{"partial overlap", enqueuedAt.Add(100 * time.Millisecond), enqueuedAt.Add(100 * time.Millisecond)},
		{"still processing", time.Time{}, publishAt},
		{"ended during publish", publishAt.Add(time.Millisecond), publishAt},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exp := setupKafkaTest(t)
			_, parent := Tracer().Start(t.Context(), "processing", trace.WithTimestamp(enqueuedAt.Add(-time.Second)))
			SetRHAttributes(parent, "buffer-org", "buffer-request")
			if !tc.processedAt.IsZero() {
				parent.End(trace.WithTimestamp(tc.processedAt))
			}
			RecordBufferWait(parent, topic, enqueuedAt, publishAt)
			if tc.processedAt.IsZero() {
				assert.True(t, parent.IsRecording())
			} else {
				assert.False(t, parent.IsRecording())
			}
			spans := exp.GetSpans()
			require.NotEmpty(t, spans)
			wait := spans[len(spans)-1]
			assert.Equal(t, "buffer wait "+topic, wait.Name)
			assert.Equal(t, tc.wantStart, wait.StartTime)
			assert.Equal(t, publishAt, wait.EndTime)
			assert.Equal(t, parent.SpanContext().SpanID(), wait.Parent.SpanID())
			assert.Equal(t, parent.SpanContext().TraceID(), wait.SpanContext.TraceID())
			assert.Equal(t, trace.SpanKindInternal, wait.SpanKind)
			assert.Equal(t, topic, attr(wait, "messaging.destination.name"))
			assert.Equal(t, "buffer-org", attr(wait, "rh.org_id"))
			assert.Equal(t, "buffer-request", attr(wait, "rh.request_id"))
			parent.End()
		})
	}
}

func TestRecordBufferWaitDisabledOrMissingLocalSpan(t *testing.T) {
	exp := setupKafkaTest(t)
	now := time.Now()
	RecordBufferWait(trace.SpanFromContext(context.Background()), "evaluation", now, now)
	remote := ContextFromTraceparent("00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01")
	RecordBufferWait(trace.SpanFromContext(remote), "evaluation", now, now)
	assert.Empty(t, exp.GetSpans())

	_, parent := Tracer().Start(t.Context(), "processing")
	t.Setenv("OTEL_MQ_ENABLED", "false")
	RecordBufferWait(parent, "evaluation", now, now)
	assert.Empty(t, exp.GetSpans())
	assert.True(t, parent.IsRecording())
	parent.End()
}
