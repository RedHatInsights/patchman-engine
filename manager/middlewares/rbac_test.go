package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler(c *gin.Context) {
	c.JSON(http.StatusOK, nil)
}

func testRBAC(t *testing.T, method string, expectedStatus int) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, "/", nil)
	router := gin.Default()
	router.Use(RBAC())
	router.Handle(method, "/", okHandler)
	router.ServeHTTP(w, req)
	assert.Equal(t, expectedStatus, w.Code)
}

func TestRBACGet(t *testing.T) {
	testRBAC(t, "GET", http.StatusOK)
}

func TestRBACPost(t *testing.T) {
	testRBAC(t, "POST", http.StatusOK)
}

func TestRBACDelete(t *testing.T) {
	testRBAC(t, "DELETE", http.StatusUnauthorized)
}

func TestRBACPut(t *testing.T) {
	testRBAC(t, "PUT", http.StatusUnauthorized)
}
