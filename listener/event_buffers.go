package listener

import (
	"app/base"
	"app/base/mqueue"
	"app/base/utils"
	"sync"
	"time"
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
	inventoryID string,
	rhAccountID int,
	ptEvent *mqueue.PayloadTrackerEvent,
) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-eval-events"))

	b.lock.Lock()
	evalData := mqueue.EvalData{
		InventoryID: inventoryID,
		RhAccountID: rhAccountID,
		OrgID:       ptEvent.OrgID,
		RequestID:   *ptEvent.RequestID,
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
	err := mqueue.SendMessages(base.Context, *b.evalWriter, b.evalBuffer)
	if err != nil {
		utils.LogError("err", err.Error(), ErrorKafkaSend)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-evaluator"))
	err = mqueue.SendMessages(base.Context, *b.ptWriter, b.ptBuffer)
	if err != nil {
		utils.LogWarn("err", err.Error(), WarnPayloadTracker)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-payload-tracker"))
	utils.LogDebug("evaluator_messages", len(b.evalBuffer),
		"payload_tracker_messages", len(b.ptBuffer), "flushed buffers")
	// empty buffer
	b.evalBuffer = b.evalBuffer[:0]
	b.ptBuffer = b.ptBuffer[:0]
}
