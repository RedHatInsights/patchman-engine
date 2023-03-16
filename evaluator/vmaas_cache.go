package evaluator

import (
	"app/base/types"
	"app/base/utils"
	"app/base/vmaas"
	"app/tasks/vmaas_sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

var memoryVmaasCache *VmaasCache

type VmaasCache struct {
	enabled       bool
	size          int
	validity      *types.Rfc3339TimestampWithZ
	checkDuration time.Duration
	data          *lru.TwoQueueCache
}

func NewVmaasPackageCache(enabled bool, size int, checkDuration time.Duration) *VmaasCache {
	c := new(VmaasCache)

	c.enabled = enabled
	c.size = size
	c.validity = vmaas_sync.GetLastSync(vmaas_sync.VmaasExported)
	c.checkDuration = checkDuration
	vmaasCacheGauge.Set(0)

	if c.enabled {
		var err error
		c.data, err = lru.New2Q(c.size)
		if err != nil {
			panic(err)
		}
		return c
	}
	return nil
}

func (c *VmaasCache) Get(checksum *string) (*vmaas.UpdatesV2Response, bool) {
	if c.enabled && checksum != nil {
		val, ok := c.data.Get(checksum)
		if ok {
			vmaasCacheCnt.WithLabelValues("hit").Inc()
			utils.LogTrace("checksum", checksum, "VmaasCache.Get cache hit")
			response := val.(*vmaas.UpdatesV2Response)
			return response, true
		}
	}
	vmaasCacheCnt.WithLabelValues("miss").Inc()
	return nil, false
}

func (c *VmaasCache) Add(checksum *string, response *vmaas.UpdatesV2Response) {
	if c.enabled && checksum != nil {
		vmaasCacheGauge.Inc()
		c.data.Add(checksum, response)
	}
}

func (c *VmaasCache) Reset(ts *types.Rfc3339TimestampWithZ) {
	c.data.Purge()
	c.validity = ts
	vmaasCacheGauge.Set(0)
}

func (c *VmaasCache) CheckValidity() {
	for range time.Tick(c.checkDuration) {
		lastModifiedTS := vmaas_sync.GetLastSync(vmaas_sync.VmaasExported)
		if lastModifiedTS == nil || c.validity == nil || c.validity.Time().Before(*lastModifiedTS.Time()) {
			c.Reset(lastModifiedTS)
		}
	}
}
