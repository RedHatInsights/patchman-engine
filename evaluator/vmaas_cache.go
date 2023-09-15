package evaluator

import (
	"app/base/types"
	"app/base/utils"
	"app/base/vmaas"
	"app/tasks/vmaas_sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

var memoryVmaasCache *VmaasCache

type VmaasCache struct {
	enabled       bool
	size          int
	currentSize   int
	validity      *types.Rfc3339TimestampWithZ
	checkDuration time.Duration
	data          *lru.TwoQueueCache[string, *vmaas.UpdatesV3Response]
}

func NewVmaasPackageCache(enabled bool, size int, checkDuration time.Duration) *VmaasCache {
	c := new(VmaasCache)

	c.enabled = enabled
	c.size = size
	c.currentSize = 0
	c.validity = vmaas_sync.GetLastSync(vmaas_sync.VmaasExported)
	c.checkDuration = checkDuration
	vmaasCacheGauge.Set(0)

	if c.enabled {
		var err error
		c.data, err = lru.New2Q[string, *vmaas.UpdatesV3Response](c.size)
		if err != nil {
			panic(err)
		}
		return c
	}
	return c
}

func (c *VmaasCache) Get(checksum *string) (*vmaas.UpdatesV3Response, bool) {
	if c.enabled && checksum != nil {
		val, ok := c.data.Get(*checksum)
		if ok {
			vmaasCacheCnt.WithLabelValues("hit").Inc()
			utils.LogTrace("checksum", *checksum, "VmaasCache.Get cache hit")
			return val, true
		}
	}
	vmaasCacheCnt.WithLabelValues("miss").Inc()
	return nil, false
}

func (c *VmaasCache) Add(checksum *string, response *vmaas.UpdatesV3Response) {
	if c.enabled && checksum != nil {
		c.data.Add(*checksum, response)
		if c.currentSize <= c.size {
			c.currentSize++
			vmaasCacheGauge.Inc()
		}
	}
}

func (c *VmaasCache) Reset(ts *types.Rfc3339TimestampWithZ) {
	if c.enabled {
		c.data.Purge()
		c.validity = ts
		vmaasCacheGauge.Set(0)
	}
}

func (c *VmaasCache) CheckValidity() {
	for range time.Tick(c.checkDuration) {
		lastModifiedTS := vmaas_sync.GetLastSync(vmaas_sync.VmaasExported)
		if lastModifiedTS == nil || c.validity == nil || c.validity.Time().Before(*lastModifiedTS.Time()) {
			c.Reset(lastModifiedTS)
		}
	}
}
