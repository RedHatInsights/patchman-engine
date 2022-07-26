package controllers

import (
	"app/base/core"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

//nolint: dupl
func TestPackageVersions(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/firefox/versions", nil, nil, PackageVersionsListHandler, 3,
		"GET", "/:package_name/versions")

	var output PackageVersionsResponse
	assert.Greater(t, len(w.Body.Bytes()), 0)
	ParseResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "76.0.1-1.fc31.x86_64", output.Data[0].Evra)
}

func TestPackageVersionsInvalidName(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/not-existing/versions", nil, nil, PackageVersionsListHandler, 3,
		"GET", "/:package_name/versions")

	assert.Equal(t, http.StatusNotFound, w.Code)
}
