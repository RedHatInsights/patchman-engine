package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthRoute(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	initRouter(HealthHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestHealthDBRouteFail(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	err := database.Db.Close()
	if err != nil { panic(err) }

	err = database.Db.Close()
	if err != nil { panic(err) }

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	initRouter(HealthDBHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHealthDBRouteOK(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	database.Configure()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	initRouter(HealthDBHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
