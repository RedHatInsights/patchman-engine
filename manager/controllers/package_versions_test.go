package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

//nolint: dupl
func TestPackageVersions(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/firefox/versions", nil)
	core.InitRouterWithParams(PackageVersionsListHandler, 3, "GET", "/:package_name/versions").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output PackageVersionsResponse
	assert.Greater(t, len(w.Body.Bytes()), 0)
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "76.0.1-1.fc31.x86_64", output.Data[0].Evra)
}

func TestPackageVersionsInvalidName(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/not-existing/versions", nil)
	core.InitRouterWithParams(PackageVersionsListHandler, 3, "GET", "/:package_name/versions").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
