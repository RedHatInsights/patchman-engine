package core

import (
	"app/base/database"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func InitRouter(handler gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	router.GET("/", handler)
	return router
}

func TestLiveness(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	InitRouter(Liveness).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReadiness(t *testing.T) {
	SetupTest(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	InitRouter(Readiness).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReadinessFail(t *testing.T) {
	SetupTest(t)

	sqlDB, _ := database.TestDB().DB()
	assert.Nil(t, sqlDB.Close())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	InitRouter(Readiness).ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
