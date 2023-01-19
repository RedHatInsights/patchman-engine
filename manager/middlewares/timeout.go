package middlewares

import (
	"app/base/utils"
	"net/http"
	"time"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
)

func WithTimeout(seconds time.Duration) gin.HandlerFunc {
	return timeout.New(
		timeout.WithTimeout(seconds*time.Second),
		timeout.WithHandler(func(c *gin.Context) {
			c.Next()
		}),
		timeout.WithResponse(func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusRequestTimeout, utils.ErrorResponse{Error: "Request timeout"})
			c.Done()
		}),
	)
}
