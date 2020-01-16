package middlewares

import (
	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"strings"
)

// Create and configure Prometheus middleware to expose metrics
func Prometheus() *ginprometheus.Prometheus {
	prometheus := ginprometheus.NewPrometheus("patchman_engine")
	unifyParametrizedUrlsCounters(prometheus)
	return prometheus
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
