package evaluator

import (
	"app/base"
	"app/base/core"
	"app/base/utils"
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	evaluationCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "How many systems were evaluated with which result",
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "evaluation",
	}, []string{"type"})

	updatesCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "How many updates were found of which type",
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "updates",
	}, []string{"type"})

	evaluationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Help:      "How long it took system evaluation of which type (upload, recalc)",
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "evaluation_duration_seconds",
	}, []string{"type"})

	evaluationPartDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Help:      "How long it took particular evaluation part",
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "evaluation_part_duration_seconds",
	}, []string{"part"})

	uploadEvaluationDelay = prometheus.NewHistogram(prometheus.HistogramOpts{
		Help:      "How long it takes from upload to evaluation",
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "upload_evaluation_delay_seconds",
		Buckets:   []float64{1, 5, 15, 30, 60, 300},
	})

	twoEvaluationsInterval = prometheus.NewHistogram(prometheus.HistogramOpts{
		Help:      "How long it takes between two evaluations",
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "two_evaluations_interval_hours",
		Buckets:   []float64{1, 2, 6, 24, 72, 168},
	})

	packageCacheCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "How many packages hit/miss package cache",
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "package_cache",
	}, []string{"type", "by"})

	packageCacheGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help:      "Package cache size",
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "package_cache_size",
	}, []string{"by"})

	vmaasCacheCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "How many vmaas checksums hit/miss cache",
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "vmaas_cache",
	}, []string{"type"})

	vmaasCacheGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Help:      "VMaaS cache size",
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "vmaas_cache_size",
	})
)

func RunMetrics() {
	prometheus.MustRegister(evaluationCnt, updatesCnt, evaluationDuration, evaluationPartDuration,
		uploadEvaluationDelay, twoEvaluationsInterval, packageCacheCnt, packageCacheGauge,
		vmaasCacheCnt, vmaasCacheGauge)

	// create web app
	app := gin.New()
	core.InitProbes(app)
	middlewares.Prometheus().Use(app)

	go base.TryExposeOnMetricsPort(app)

	publicPort := utils.CoreCfg.PublicPort
	err := utils.RunServer(base.Context, app, publicPort)
	if err != nil {
		utils.LogError("err", err.Error())
		panic(err)
	}
}
