package core

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func createSleepHandler(duration time.Duration) gin.HandlerFunc {
	handler := func(c *gin.Context) {
		time.Sleep(duration)
		c.JSON(http.StatusOK, "ok")
	}
	return handler
}

func TestTimeoutError(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	InitRouterWithTimeout(createSleepHandler(time.Second), 1).ServeHTTP(w, req)
	assert.Equal(t, http.StatusRequestTimeout, w.Code)
}

func TestTimeoutOK(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	InitRouterWithTimeout(createSleepHandler(0), 1).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
