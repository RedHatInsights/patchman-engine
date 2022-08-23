package middlewares

import (
	"app/base/utils"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

var serviceErrorCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "patchman_engine",
	Subsystem: "manager",
	Name:      "dependency_call",
}, []string{"name", "status"})

var requestDurations = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Help:      "Request Durations",
	Namespace: "patchman_engine",
	Subsystem: "manager",
	Name:      "request_durations",
	Buckets:   []float64{1, 1.5, 1.75, 2, 2.5, 3, 3.5, 4},
}, []string{"endpoint"})

// Create and configure Prometheus middleware to expose metrics
func Prometheus() *ginprometheus.Prometheus {
	prometheus.MustRegister(serviceErrorCnt, requestDurations)

	p := ginprometheus.NewPrometheus("patchman_engine")
	p.MetricsPath = utils.Cfg.MetricsPath
	unifyParametrizedUrlsCounters(p)
	return p
}

func unifyParametrizedUrlsCounters(p *ginprometheus.Prometheus) {
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		url := c.Request.URL.Path
		for _, p := range c.Params {
			url = strings.Replace(url, "/"+p.Value, "/:"+p.Key, 1)
		}
		return url
	}
}
