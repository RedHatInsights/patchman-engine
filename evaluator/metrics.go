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
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "evaluation",
	}, []string{"type"})

	updatesCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "updates",
	}, []string{"type"})

	evaluationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "evaluation_duration_seconds",
	}, []string{"type"})

	evaluationPartDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
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
)

func RunMetrics(port string) {
	prometheus.MustRegister(evaluationCnt, updatesCnt, evaluationDuration, evaluationPartDuration,
		uploadEvaluationDelay)

	// create web app
	app := gin.New()
	core.InitProbes(app)
	middlewares.Prometheus().Use(app)

	err := utils.RunServer(base.Context, app, ":"+port)
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}
