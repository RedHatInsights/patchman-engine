package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/?filter[system_profile][is_sap][eq]=true", nil)
	core.InitRouterWithParams(PackagesListHandler, 3, "GET", "/").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output PackagesResponse
	assert.Greater(t, len(w.Body.Bytes()), 0)
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, "kernel", output.Data[3].Name)
	assert.Equal(t, 2, output.Data[3].SystemsInstalled)
	assert.Equal(t, 1, output.Data[3].SystemsUpdatable)
}

func TestSearchPackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/?search=fire", nil)
	core.InitRouterWithParams(PackagesListHandler, 3, "GET", "/").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output PackagesResponse
	assert.Greater(t, len(w.Body.Bytes()), 0)
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "firefox", output.Data[0].Name)
}
