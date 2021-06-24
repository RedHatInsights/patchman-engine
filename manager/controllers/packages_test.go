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

func doTestPackages(t *testing.T, q string) PackagesResponse {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", q, nil)
	core.InitRouterWithParams(PackagesListHandler, 3, "GET", "/").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output PackagesResponse
	assert.Greater(t, len(w.Body.Bytes()), 0)
	ParseReponseBody(t, w.Body.Bytes(), &output)
	return output
}

func TestPackagesFilterInstalled(t *testing.T) {
	output := doTestPackages(t, "/?filter[systems_installed]=44")
	assert.Equal(t, 0, len(output.Data))
}

func TestPackagesFilterUpdatable(t *testing.T) {
	output := doTestPackages(t, "/?filter[systems_updatable]=4")
	assert.Equal(t, 0, len(output.Data))
}

func TestPackagesFilterSummary(t *testing.T) {
	output := doTestPackages(t, `/?filter[summary]=Mozilla Firefox Web browser`)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "firefox", output.Data[0].Name)
	assert.Equal(t, 2, output.Data[0].SystemsInstalled)
	assert.Equal(t, 2, output.Data[0].SystemsUpdatable)
}

func TestPackagesFilterSAP(t *testing.T) {
	output := doTestPackages(t, "/?filter[system_profile][is_sap][eq]=true")
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, "kernel", output.Data[3].Name)
	assert.Equal(t, 2, output.Data[3].SystemsInstalled)
	assert.Equal(t, 1, output.Data[3].SystemsUpdatable)
}

func TestSearchPackages(t *testing.T) {
	output := doTestPackages(t, "/?search=fire")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "firefox", output.Data[0].Name)
}

func TestPackageTagsInvalid(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k3=val4&tags=invalidTag", nil)
	core.InitRouterWithParams(PackagesListHandler, 3, "GET", "/").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestPackagesWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/", "/?offset=1000", PackagesListHandler)
}
