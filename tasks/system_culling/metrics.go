package system_culling //nolint:revive,stylecheck

import (
	"app/base/utils"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	deletedCulledSystemsCnt = prometheus.NewCounter(prometheus.CounterOpts{
		Help:      "How many culled systems were deleted",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "deleted_culled_systems",
	})
	staleSystemsMarkedCnt = prometheus.NewCounter(prometheus.CounterOpts{
		Help:      "How many systems were marked as stale",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "stale_systems_marked",
	})
)

func Metrics() *push.Pusher {
	registry := prometheus.NewRegistry()
	registry.MustRegister(deletedCulledSystemsCnt, staleSystemsMarkedCnt)
	pusher := push.New(utils.Cfg.PrometheusPushGateway, "system_culling").Gatherer(registry)
	return pusher
}
