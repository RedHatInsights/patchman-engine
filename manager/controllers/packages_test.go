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

func doTestPackagesBytes(t *testing.T, q string) (resp []byte, status int) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", q, nil)
	core.InitRouterWithParams(PackagesListHandler, 3, "GET", "/").
		ServeHTTP(w, req)
	return w.Body.Bytes(), w.Code
}

func doTestPackages(t *testing.T, q string) PackagesResponse {
	respBytes, code := doTestPackagesBytes(t, q)
	assert.Equal(t, http.StatusOK, code)
	var output PackagesResponse
	assert.Greater(t, len(respBytes), 0)
	ParseResponseBody(t, respBytes, &output)
	return output
}

func TestPackagesFilterInstalled(t *testing.T) {
	output := doTestPackages(t, "/?filter[systems_installed]=44")
	assert.Equal(t, 0, len(output.Data))
}

func TestPackagesEmptyResponse(t *testing.T) {
	respBytes, code := doTestPackagesBytes(t, "/?filter[systems_installed]=44")
	assert.Equal(t, http.StatusOK, code)
	respStr := string(respBytes)
	assert.Equal(t, "{\"data\":[]", respStr[:10])
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
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestPackagesWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/", "/?offset=1000", PackagesListHandler)
}

func TestPackageTagsInMetadata(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k3=val4&tags=ns1/k1=val1", nil)
	core.InitRouterWithParams(PackagesListHandler, 3, "GET", "/").ServeHTTP(w, req)

	var output PackagesResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)

	testMap := map[string]FilterData{
		"ns1/k1": {"eq", []string{"val1"}},
		"ns1/k3": {"eq", []string{"val4"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}
