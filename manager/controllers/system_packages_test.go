package controllers

import (
	"app/base/core"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemPackages(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/:inventory_id/packages", "00000000-0000-0000-0000-000000000013", "",
		nil, "", SystemPackagesHandler, 3)

	var output SystemPackageResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Len(t, output.Data, 4)
	assert.Equal(t, output.Data[0].Name, "bash")
	assert.Equal(t, len(output.Data[0].Updates), 0)
	assert.Equal(t, output.Data[1].Name, "curl")
	assert.Equal(t, len(output.Data[1].Updates), 0)
	assert.Equal(t, output.Data[2].Name, "firefox")
	assert.Equal(t, len(output.Data[2].Updates), 2)
	assert.Equal(t, output.Data[3].Name, "kernel")
	assert.Equal(t, len(output.Data[3].Updates), 0)
	assert.Equal(t, output.Meta.TotalItems, 4)
}

func TestPackagesSearch(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/:inventory_id/packages", "00000000-0000-0000-0000-000000000012",
		"?search=kernel", nil, "", SystemPackagesHandler, 3)

	var output SystemPackageResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Len(t, output.Data, 1)
	assert.Equal(t, output.Data[0].Name, "kernel")
}

func TestNoPackages(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/:inventory_id/packages", "00000000-0000-0000-0000-000000000001", "",
		nil, "", SystemPackagesHandler, 1)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSystemPackagesUpdatableOnly(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/:inventory_id/packages", "00000000-0000-0000-0000-000000000013",
		"?filter[updatable]=true", nil, "", SystemPackagesHandler, 3)

	var output SystemPackageResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Len(t, output.Data, 1)
	assert.Equal(t, output.Data[0].Name, "firefox")
}

func TestSystemPackagesName(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/:inventory_id/packages", "00000000-0000-0000-0000-000000000013",
		"?filter[name]=firefox", nil, "", SystemPackagesHandler, 3)

	var output SystemPackageResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Len(t, output.Data, 1)
	assert.Equal(t, output.Data[0].Name, "firefox")
}

func TestSystemPackagesNonUpdatableOnly(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/:inventory_id/packages", "00000000-0000-0000-0000-000000000013",
		"?filter[updatable]=false", nil, "", SystemPackagesHandler, 3)

	var output SystemPackageResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Len(t, output.Data, 3)
	assert.Equal(t, output.Data[0].Name, "bash")
	assert.Equal(t, output.Data[1].Name, "curl")
	assert.Equal(t, output.Data[2].Name, "kernel")
}

func TestSystemPackagesWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/:inventory_id/packages",
		"00000000-0000-0000-0000-000000000001", "?offset=1000", SystemPackagesHandler)
}

func TestSystemPackagesUnknown(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/:inventory_id/packages", "unknownsystem", "", nil, "",
		SystemPackagesHandler, 3)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
