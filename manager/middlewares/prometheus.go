package middlewares

import (
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"strings"
)

var serviceErrorCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "patchman_engine",
	Subsystem: "manager",
	Name:      "dependency_call",
}, []string{"name", "status"})

// Create and configure Prometheus middleware to expose metrics
func Prometheus() *ginprometheus.Prometheus {
	prometheus.MustRegister(serviceErrorCnt)

	p := ginprometheus.NewPrometheus("patchman_engine")
	p.MetricsPath = utils.Getenv("METRICS_PATH", "/metrics")
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
