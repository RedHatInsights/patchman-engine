package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func LimitRequestBodySize(size int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request != nil && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, size)
		}
		c.Next()
	}
}
