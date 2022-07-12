package controllers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

//nolint: dupl
func TestPackageVersions(t *testing.T) {
	SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/firefox/versions", nil, nil, PackageVersionsListHandler, 3,
		"GET", "/:package_name/versions")

	assert.Equal(t, http.StatusOK, w.Code)
	var output PackageVersionsResponse
	assert.Greater(t, len(w.Body.Bytes()), 0)
	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "76.0.1-1.fc31.x86_64", output.Data[0].Evra)
}

func TestPackageVersionsInvalidName(t *testing.T) {
	SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/not-existing/versions", nil, nil, PackageVersionsListHandler, 3,
		"GET", "/:package_name/versions")

	assert.Equal(t, http.StatusNotFound, w.Code)
}
