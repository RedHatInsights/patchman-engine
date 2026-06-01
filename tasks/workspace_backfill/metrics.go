package workspace_backfill

import (
	"app/base/utils"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	backfillRowsCnt = prometheus.NewCounter(prometheus.CounterOpts{
		Help:      "How many system_inventory rows were backfilled with workspace_id and workspace_name",
		Namespace: "patchman_engine",
		Subsystem: "workspace_backfill",
		Name:      "rows_updated",
	})
	backfillBatchesCnt = prometheus.NewCounter(prometheus.CounterOpts{
		Help:      "How many workspace backfill batches completed successfully",
		Namespace: "patchman_engine",
		Subsystem: "workspace_backfill",
		Name:      "batches",
	})
	backfillErrorsCnt = prometheus.NewCounter(prometheus.CounterOpts{
		Help:      "How many workspace backfill batches failed",
		Namespace: "patchman_engine",
		Subsystem: "workspace_backfill",
		Name:      "batch_errors",
	})
)

// Metrics returns a pushgateway pusher for workspace backfill counters.
func Metrics() *push.Pusher {
	registry := prometheus.NewRegistry()
	registry.MustRegister(backfillRowsCnt, backfillBatchesCnt, backfillErrorsCnt)
	return push.New(utils.CoreCfg.PrometheusPushGateway, "workspace_backfill").Gatherer(registry)
}
