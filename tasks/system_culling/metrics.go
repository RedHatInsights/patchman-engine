package system_culling //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/core"
	"app/base/utils"
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
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

func RunMetrics() {
	prometheus.MustRegister(deletedCulledSystemsCnt, staleSystemsMarkedCnt)

	// create web app
	app := gin.New()
	core.InitProbes(app)
	middlewares.Prometheus().Use(app)

	go base.TryExposeOnMetricsPort(app)

	port := utils.Cfg.PublicPort
	err := utils.RunServer(base.Context, app, port)
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}
