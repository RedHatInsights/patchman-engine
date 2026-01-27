package listener

import (
	"app/base"
	"app/base/mqueue"
	"app/base/utils"
	"sync"
	"time"
)

// accumulate events and create group PlatformEvents to save some resources
var eventBufferSize = 5 * mqueue.BatchSize

type eventBuffer struct {
	EvalBuffer mqueue.EvalDataSlice
	PtBuffer   mqueue.PayloadTrackerEvents
	Lock       sync.Mutex
	flushTimer *time.Timer
}

var updatedEventsBuffer = eventBuffer{
	EvalBuffer: make(mqueue.EvalDataSlice, 0, eventBufferSize+1),
	PtBuffer:   make(mqueue.PayloadTrackerEvents, 0, eventBufferSize+1),
	Lock:       sync.Mutex{},
}

var createdEventsBuffer = eventBuffer{
	EvalBuffer: make(mqueue.EvalDataSlice, 0, eventBufferSize+1),
	PtBuffer:   make(mqueue.PayloadTrackerEvents, 0, eventBufferSize+1),
	Lock:       sync.Mutex{},
}

// var flushTimer = time.AfterFunc(87600*time.Hour, func() {
// 	utils.LogInfo(FlushedTimeoutBuffer)
// 	updatedEventsBuffer.flushEvalEvents()
// })

func (b *eventBuffer) initFlushTimer(w *mqueue.Writer) {
	b.flushTimer = time.AfterFunc(87600*time.Hour, func() {
		utils.LogInfo(FlushedTimeoutBuffer)
		b.flushEvalEvents(w)
	})
}

// send events after full buffer or timeout
func (b *eventBuffer) bufferEvalEvents(
	inventoryID string,
	rhAccountID int,
	ptEvent *mqueue.PayloadTrackerEvent,
	w *mqueue.Writer,
) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-eval-events"))

	b.Lock.Lock()
	evalData := mqueue.EvalData{
		InventoryID: inventoryID,
		RhAccountID: rhAccountID,
		OrgID:       ptEvent.OrgID,
		RequestID:   *ptEvent.RequestID,
	}
	b.EvalBuffer = append(b.EvalBuffer, evalData)
	b.PtBuffer = append(b.PtBuffer, *ptEvent)
	b.Lock.Unlock()

	b.flushTimer.Reset(uploadEvalTimeout)
	if len(b.EvalBuffer) >= eventBufferSize {
		utils.LogInfo(FlushedFullBuffer)
		b.flushEvalEvents(w)
	}
}

func (b *eventBuffer) flushEvalEvents(w *mqueue.Writer) {
	tStart := time.Now()
	b.Lock.Lock()
	defer b.Lock.Unlock()
	err := mqueue.SendMessages(base.Context, *w, b.EvalBuffer)
	if err != nil {
		utils.LogError("err", err.Error(), ErrorKafkaSend)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-evaluator"))
	err = mqueue.SendMessages(base.Context, ptWriter, b.PtBuffer)
	if err != nil {
		utils.LogWarn("err", err.Error(), WarnPayloadTracker)
	}
	utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("buffer-sent-payload-tracker"))
	utils.LogDebug("evaluator_messages", len(b.EvalBuffer),
		"payload_tracker_messages", len(b.PtBuffer), "flushed buffers")
	// empty buffer
	b.EvalBuffer = b.EvalBuffer[:0]
	b.PtBuffer = b.PtBuffer[:0]
}
