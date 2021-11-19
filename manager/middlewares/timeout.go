package middlewares

import (
	"app/base/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func Timeout(timeoutMs int) gin.HandlerFunc {
	errMsg := fmt.Sprintf("Request timeout %d ms", timeoutMs)
	return func(c *gin.Context) {
		// https://gobyexample.com/timeouts
		pipe := make(chan bool, 1)
		go func() {
			c.Next()
			pipe <- true
		}()

		select {
		// Handler function ended in time.
		case <-pipe:
			return

		// Timeout, handler function interrupted.
		case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
			c.AbortWithStatusJSON(http.StatusRequestTimeout, utils.ErrorResponse{Error: errMsg})
			return
		}
	}
}
