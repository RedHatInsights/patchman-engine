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
	req, _ := http.NewRequest("GET", "/INV-12/packages", nil)
	core.InitRouterWithParams(SystemPackagesHandler, "3", "GET", "/:inventory_id/packages").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemPackageResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Contains(t, output[0].Name, "kernel")
	assert.Contains(t, output[1].Name, "firefox")
	assert.Len(t, output[0].Updates, 1)
	assert.Len(t, output[1].Updates, 2)
}

func TestNoPackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/INV-1/packages", nil)
	core.InitRouterWithParams(SystemPackagesHandler, "1", "GET", "/:inventory_id/packages").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
