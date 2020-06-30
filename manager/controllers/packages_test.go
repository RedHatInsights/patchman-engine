package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
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
	assert.Equal(t, output.Meta.TotalItems, 2)

	assert.Contains(t, output.Data[0].Name, "firefox")
	assert.Contains(t, output.Data[1].Name, "kernel")
	assert.Len(t, output.Data[0].Updates, 2)
	assert.Len(t, output.Data[1].Updates, 1)
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

func TestPackagesFilter(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	for name := range PackageOpts.Fields {

		w := httptest.NewRecorder()
		path := fmt.Sprintf("/INV-12/packages?filter[%s]=eq:2", name)
		req, _ := http.NewRequest("GET", path, nil)
		core.InitRouterWithParams(SystemPackagesHandler, "3", "GET", "/:inventory_id/packages").
			ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestPackagesSearch(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/INV-12/packages?search=fire", nil)
	core.InitRouterWithParams(SystemPackagesHandler, "3", "GET", "/:inventory_id/packages").
		ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemPackageResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, output.Meta.TotalItems, 1)
}
