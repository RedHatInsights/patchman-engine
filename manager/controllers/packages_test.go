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

	assert.Equal(t, 200, w.Code)
	var output SystemPackageResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Len(t, output.Data, 2)
	assert.Equal(t, output.Data[0].Name, "firefox")
	assert.Equal(t, output.Data[1].Name, "kernel")
}

func TestNoPackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/INV-1/packages", nil)
	core.InitRouterWithParams(SystemPackagesHandler, "1", "GET", "/:inventory_id/packages").
		ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, 200, w.Code)
}
