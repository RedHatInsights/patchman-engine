package middlewares

import (
	"app/base/utils"
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

func LimitRequestHeaders(maxHeaderCount int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(c.Request.Header) > maxHeaderCount {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, utils.ErrorResponse{Error: "too many headers"})
		}
	}
}
