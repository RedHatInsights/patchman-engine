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
)

func init() {
	prometheus.MustRegister(evaluationCnt, updatesCnt)
}
