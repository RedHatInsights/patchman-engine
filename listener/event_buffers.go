package listener

import (
	"app/base"
	"app/base/mqueue"
	"app/base/telemetry"
	"app/base/utils"
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type bufferedTrace struct {
	parent     trace.Span
	enqueuedAt time.Time
}

type eventBuffer struct {
	evalBuffer mqueue.EvalDataSlice
	ptBuffer   mqueue.PayloadTrackerEvents
	traces     []bufferedTrace
	lock       sync.Mutex
	flushTimer *time.Timer
	evalWriter *mqueue.Writer
	ptWriter   *mqueue.Writer
	evalTopic  string
}

func (b *eventBuffer) initEventBuffer(evalWriter, ptWriter *mqueue.Writer, evalTopic string) {
	b.evalBuffer = make(mqueue.EvalDataSlice, 0, eventBufferSize+1)
	b.ptBuffer = make(mqueue.PayloadTrackerEvents, 0, eventBufferSize+1)
	b.traces = nil
	b.lock = sync.Mutex{}
	b.flushTimer = time.AfterFunc(87600*time.Hour, func() {
		utils.LogInfo(FlushedTimeoutBuffer)
		b.flushEvalEvents()
	})
	b.evalWriter = evalWriter
	b.ptWriter = ptWriter
	b.evalTopic = evalTopic
}

// send events after full buffer or timeout
func (b *eventBuffer) bufferEvalEvents(
	inventoryID uuid.UUID,
	rhAccountID int,
	ptEvent *mqueue.PayloadTrackerEvent,
	ctx context.Context,
) {
	defer utils.ObserveSecondsSince(time.Now(), messagePartDuration.WithLabelValues("buffer-eval-events"))

	b.lock.Lock()
	evalData := mqueue.EvalData{
		InventoryID: inventoryID,
		RhAccountID: rhAccountID,
		OrgID:       ptEvent.OrgID,
		RequestID:   *ptEvent.RequestID,
		Traceparent: telemetry.EncodeTraceparent(ctx),
	}
	b.evalBuffer = append(b.evalBuffer, evalData)
	b.ptBuffer = append(b.ptBuffer, *ptEvent)
	if telemetry.MQEnabled() {
		b.traces = append(b.traces, bufferedTrace{parent: trace.SpanFromContext(ctx), enqueuedAt: time.Now()})
	}

	b.flushTimer.Reset(uploadEvalTimeout)
	shouldFlush := len(b.evalBuffer) >= eventBufferSize
	b.lock.Unlock()

	if shouldFlush {
		utils.LogInfo(FlushedFullBuffer)
		b.flushEvalEvents()
	}
}

func (b *eventBuffer) flushEvalEvents() {
	tStart := time.Now()
	b.lock.Lock()
	defer b.lock.Unlock()
	if len(b.evalBuffer) == 0 {
		return
	}
	b.flushTimer.Stop()

	tps := make([]string, 0, len(b.evalBuffer))
	for _, e := range b.evalBuffer {
		tps = append(tps, e.Traceparent)
	}
	publishAt := time.Now()
	pctx, span := telemetry.ProducerContext(
		telemetry.ParentContextFromTraceparents(tps),
		b.evalTopic,
		nil,
	)
	err := mqueue.SendMessages(pctx, *b.evalWriter, b.evalBuffer)
	telemetry.End(span, err)
	if err != nil {
		utils.LogError("err", err, ErrorKafkaSend)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-evaluator"))
	for _, buffered := range b.traces {
		telemetry.RecordBufferWait(buffered.parent, b.evalTopic, buffered.enqueuedAt, publishAt)
	}
	if ptErr := mqueue.SendMessages(base.Context, *b.ptWriter, b.ptBuffer); ptErr != nil {
		utils.LogWarn("err", ptErr, WarnPayloadTracker)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-payload-tracker"))
	utils.LogDebug("evaluator_messages", len(b.evalBuffer),
		"payload_tracker_messages", len(b.ptBuffer), "flushed buffers")
	// empty buffer
	b.evalBuffer = b.evalBuffer[:0]
	b.ptBuffer = b.ptBuffer[:0]
	clear(b.traces)
	b.traces = b.traces[:0]
}
