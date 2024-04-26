package controllers

import (
	"app/base/core"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageVersions(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/:package_name/versions", "firefox", "", nil, "",
		PackageVersionsListHandler, 3)

	var output PackageVersionsResponse
	assert.Greater(t, len(w.Body.Bytes()), 0)
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "76.0.1-1.fc31.x86_64", output.Data[0].Evra)
}

func TestPackageVersionsInvalidName(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/:package_name/versions", "not-existing", "", nil, "",
		PackageVersionsListHandler, 3)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
