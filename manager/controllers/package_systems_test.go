package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageSystems(t *testing.T) {
	output := testPackageSystems(t, "/kernel/systems?sort=id", 3)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output.Data[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output.Data[0].DisplayName)
	assert.Equal(t, "5.6.13-200.fc31.x86_64", output.Data[0].InstalledEVRA)
	assert.Equal(t, "5.10.13-200.fc31.x86_64", output.Data[0].AvailableEVRA)
	assert.Equal(t, SystemTagsList{{"k1", "ns1", "val1"}}, output.Data[0].Tags)
	assert.Equal(t, "", output.Data[0].BaselineName)
	assert.Equal(t, utils.PtrBoolNil(), output.Data[0].BaselineUpToDate)
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
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", url, nil, nil, PackageSystemsListHandler, account, "GET",
		"/:package_name/systems")

	var output PackageSystemsResponse
	ParseResponse(t, w, http.StatusOK, &output)
	return output
}

func testPackageSystemsError(t *testing.T, url string, account int) (int, utils.ErrorResponse) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", url, nil, nil, PackageSystemsListHandler, account, "GET",
		"/:package_name/systems")

	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	return w.Code, errResp
}

func TestPackageSystemsTagsInMetadata(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/kernel/systems?tags=ns1/k3=val4&tags=ns1/k1=val1", nil, nil,
		PackageSystemsListHandler, 3, "GET", "/:package_name/systems")

	var output PackageSystemsResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)

	testMap := map[string]FilterData{
		"ns1/k1": {"eq", []string{"val1"}},
		"ns1/k3": {"eq", []string{"val4"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}
