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
)

type eventBuffer struct {
	evalBuffer mqueue.EvalDataSlice
	ptBuffer   mqueue.PayloadTrackerEvents
	lock       sync.Mutex
	flushTimer *time.Timer
	evalWriter *mqueue.Writer
	ptWriter   *mqueue.Writer
}

func (b *eventBuffer) initEventBuffer(evalWriter, ptWriter *mqueue.Writer) {
	b.evalBuffer = make(mqueue.EvalDataSlice, 0, eventBufferSize+1)
	b.ptBuffer = make(mqueue.PayloadTrackerEvents, 0, eventBufferSize+1)
	b.lock = sync.Mutex{}
	b.flushTimer = time.AfterFunc(87600*time.Hour, func() {
		utils.LogInfo(FlushedTimeoutBuffer)
		b.flushEvalEvents()
	})
	b.evalWriter = evalWriter
	b.ptWriter = ptWriter
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

	tps := make([]string, 0, len(b.evalBuffer))
	for _, e := range b.evalBuffer {
		tps = append(tps, e.Traceparent)
	}
	links := telemetry.LinksFromTraceparents(tps)
	pctx, span := telemetry.ProducerContext(context.Background(), utils.CoreCfg.EvalTopic, links)
	var err error
	defer func() { telemetry.End(span, err) }()
	err = mqueue.SendMessages(pctx, *b.evalWriter, b.evalBuffer)
	if err != nil {
		utils.LogError("err", err, ErrorKafkaSend)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-evaluator"))
	if ptErr := mqueue.SendMessages(base.Context, *b.ptWriter, b.ptBuffer); ptErr != nil {
		utils.LogWarn("err", ptErr, WarnPayloadTracker)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-payload-tracker"))
	utils.LogDebug("evaluator_messages", len(b.evalBuffer),
		"payload_tracker_messages", len(b.ptBuffer), "flushed buffers")
	// empty buffer
	b.evalBuffer = b.evalBuffer[:0]
	b.ptBuffer = b.ptBuffer[:0]
}
