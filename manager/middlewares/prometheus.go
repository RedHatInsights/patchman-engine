package middlewares

import ginprometheus "github.com/zsais/go-gin-prometheus"

// Create and configure Prometheus middleware to expose metrics
func Prometheus() *ginprometheus.Prometheus{
	prometheus := ginprometheus.NewPrometheus("patchman_engine")
	return prometheus
}
