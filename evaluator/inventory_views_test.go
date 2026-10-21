package evaluator

import (
	"app/base/models"
	"app/base/mqueue"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

type inventoryViewsContextWriter struct {
	mqueue.MockKafkaWriter
	ctx context.Context
	err error
}

func (w *inventoryViewsContextWriter) WriteMessages(ctx context.Context, messages ...mqueue.KafkaMessage) error {
	w.ctx = ctx
	if w.err != nil {
		return w.err
	}
	return w.MockKafkaWriter.WriteMessages(ctx, messages...)
}

func TestInventoryViewsPreservesEvaluationContext(t *testing.T) {
	previous := inventoryViewsPublisher
	t.Cleanup(func() { inventoryViewsPublisher = previous })
	parent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(t.Context(), parent)
	systems := []models.SystemPlatformV2{{Inventory: models.SystemInventory{
		ID: 1, RhAccountID: 1, InventoryID: uuid.New(),
	}}}
	orgID := "org-1"
	origin := &mqueue.PlatformEvent{OrgID: &orgID, RequestIDs: []string{"request-1"}}

	for _, test := range []struct {
		name string
		err  error
	}{{name: "success"}, {name: "publish error", err: errors.New("Kafka unavailable")}} {
		t.Run(test.name, func(t *testing.T) {
			writer := &inventoryViewsContextWriter{err: test.err}
			inventoryViewsPublisher = writer
			err := publishInventoryViewsEvent(ctx, nil, systems, origin)
			require.ErrorIs(t, err, test.err)
			sent := trace.SpanContextFromContext(writer.ctx)
			assert.Equal(t, parent.TraceID(), sent.TraceID())
			if test.err == nil {
				require.Len(t, writer.Messages, 1)
				assert.Equal(t, []byte("org-1"), writer.Messages[0].Key)
			}
		})
	}
}
