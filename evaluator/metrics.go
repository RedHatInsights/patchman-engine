package evaluator

import (
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

	evaluationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "patchman_engine",
		Subsystem: "evaluator",
		Name:      "evaluation_duration_milliseconds",
		Buckets:   []float64{.00001, .0001, .001, .01, .1, 1, 10, 100, 1000},
	})
)

func init() {
	prometheus.MustRegister(evaluationCnt, updatesCnt, evaluationDuration)
}
