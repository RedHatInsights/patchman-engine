package listener

import (
	"gin-container/app/utils"
	"time"
)

type Benchmark struct {
	startTime      time.Time
	nItemsToWrite  int
	nWritten       int
	storage        *Storage
}

func InitBenchmark(nItemsToWrite int, storage *Storage) *Benchmark {
	benchmark := Benchmark{
		startTime:     time.Now(),
		nItemsToWrite: nItemsToWrite,
		nWritten:      0,
		storage:       storage,
	}
	return &benchmark
}

func (b *Benchmark) Increment() {
	b.nWritten += 1
	if b.nWritten == b.nItemsToWrite {
		err := b.storage.Flush()
		if err != nil {
			utils.Log("err", err.Error()).Info("cannot flush to storage to finish benchmark!!!")
			return
		}

		duration := time.Since(b.startTime).Seconds()
		utils.Log("write/sec", float64(b.nWritten) / duration, "items", b.nWritten,
			"duration", duration).Info("batch finished")
	}
}
