package caches

import (
	"app/base/utils"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var packageRefreshPartDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Help:      "How long it took particular package refresh part",
	Namespace: "patchman_engine",
	Subsystem: "caches",
	Name:      "package_refresh_part_duration_seconds",
}, []string{"part"})

func Metrics() *push.Pusher {
	registry := prometheus.NewRegistry()
	registry.MustRegister(packageRefreshPartDuration)

	return push.New(utils.Cfg.PrometheusPushGateway, "caches").Gatherer(registry)
}
