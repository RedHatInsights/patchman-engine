package middlewares

import (
	"gin-container/app/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// setup logging middleware
// ensures logging line after each http response with fields:
// duration_ms, status, userAgent, method, remoteAddr, url, param_*
func RequestResponseLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		tStart := time.Now()
		c.Next()
		var fields []interface{}

		duration := time.Now().Sub(tStart).Nanoseconds() / 1e6
		fields = append(fields, "durationMs", duration,
			"status", c.Writer.Status(),
			"userAgent", c.Request.UserAgent(),
			"method", c.Request.Method,
			"remoteAddr", c.Request.RemoteAddr,
			"url", c.Request.URL.String())

		for _, param := range c.Params {
			fields = append(fields, "param_" + param.Key, param.Value)
		}

		if c.Writer.Status() < http.StatusInternalServerError {
			utils.Log(fields...).Info("request")
		} else {
			utils.Log(fields...).Error("request")
		}
	}
}
