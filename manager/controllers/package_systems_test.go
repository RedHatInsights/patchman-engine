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

func TestPackageSystems(t *testing.T) {
	output := testPackageSystems(t, "/kernel/systems?sort=id", 3)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output.Data[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output.Data[0].DisplayName)
	assert.Equal(t, "5.6.13-200.fc31.x86_64", output.Data[0].InstalledEVRA)
	assert.Equal(t, "5.10.13-200.fc31.x86_64", output.Data[0].AvailableEVRA)
	assert.Equal(t, SystemTagsList{{"k1", "ns1", "val1"}}, output.Data[0].Tags)
}

func TestPackageSystemsWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/:package_name/systems", "/kernel/systems?offset=1000", PackageSystemsListHandler)
}

func TestPackageSystemsTagsInvalid(t *testing.T) {
	code, errResp := testPackageSystemsError(t, "/kernel/systems?tags=ns1/k3=val4&tags=invalidTag", 3)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestPackageSystemsInvalidName(t *testing.T) {
	code, errResp := testPackageSystemsError(t, "/unknown_package/systems", 3)
	assert.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, "package not found", errResp.Error)
}

func testPackageSystems(t *testing.T, url string, account int) PackageSystemsResponse {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	core.InitRouterWithParams(PackageSystemsListHandler, account, "GET", "/:package_name/systems").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output PackageSystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	return output
}

func testPackageSystemsError(t *testing.T, url string, account int) (int, utils.ErrorResponse) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	core.InitRouterWithParams(PackageSystemsListHandler, account, "GET", "/:package_name/systems").
		ServeHTTP(w, req)

	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	return w.Code, errResp
}
