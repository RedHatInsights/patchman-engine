package controllers

import (
	"app/base/core"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemPackages(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/00000000-0000-0000-0000-000000000013/packages",
		nil, nil, SystemPackagesHandler, 3, "GET", "/:inventory_id/packages")

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemPackageResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Len(t, output.Data, 4)
	assert.Equal(t, output.Data[0].Name, "bash")
	assert.Equal(t, len(output.Data[0].Updates), 0)
	assert.Equal(t, output.Data[1].Name, "curl")
	assert.Equal(t, len(output.Data[1].Updates), 0)
	assert.Equal(t, output.Data[2].Name, "firefox")
	assert.Equal(t, len(output.Data[2].Updates), 2)
	assert.Equal(t, output.Data[3].Name, "kernel")
	assert.Equal(t, len(output.Data[3].Updates), 0)
}

func TestPackagesSearch(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/00000000-0000-0000-0000-000000000012/packages?search=kernel",
		nil, nil, SystemPackagesHandler, 3, "GET", "/:inventory_id/packages")

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemPackageResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Len(t, output.Data, 1)
	assert.Equal(t, output.Data[0].Name, "kernel")
}

func TestNoPackages(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/00000000-0000-0000-0000-000000000001/packages",
		nil, nil, SystemPackagesHandler, 1, "GET", "/:inventory_id/packages")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSystemPackagesUpdatableOnly(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/00000000-0000-0000-0000-000000000013/packages?filter[updatable]=true",
		nil, nil, SystemPackagesHandler, 3, "GET", "/:inventory_id/packages")

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemPackageResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Len(t, output.Data, 1)
	assert.Equal(t, output.Data[0].Name, "firefox")
}

func TestSystemPackagesNonUpdatableOnly(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET",
		"/00000000-0000-0000-0000-000000000013/packages?filter[updatable]=false", nil, nil,
		SystemPackagesHandler, 3, "GET", "/:inventory_id/packages")

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemPackageResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Len(t, output.Data, 3)
	assert.Equal(t, output.Data[0].Name, "bash")
	assert.Equal(t, output.Data[1].Name, "curl")
	assert.Equal(t, output.Data[2].Name, "kernel")
}

func TestSystemPackagesWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/:inventory_id/packages",
		"/00000000-0000-0000-0000-000000000001/packages?offset=1000", SystemPackagesHandler)
}

func TestSystemPackagesUnknown(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/unknownsystem/packages", nil, nil,
		SystemPackagesHandler, 3, "GET", "/:inventory_id/packages")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
