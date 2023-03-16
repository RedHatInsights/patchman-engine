package controllers

import (
	"app/base/core"
	"app/base/utils"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemDetailDefault1(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/00000000-0000-0000-0000-000000000001", nil, "",
		SystemDetailHandler, "/:inventory_id")

	var output SystemDetailResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data.ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data.Attributes.DisplayName)
	assert.Equal(t, "system", output.Data.Type)
	assert.Equal(t, "2018-09-22 16:00:00 +0000 UTC", output.Data.Attributes.LastEvaluation.String())
	assert.Equal(t, "2020-09-22 16:00:00 +0000 UTC", output.Data.Attributes.LastUpload.String())
	assert.False(t, output.Data.Attributes.Stale)
	assert.True(t, output.Data.Attributes.ThirdParty)
	assert.Equal(t, 2, output.Data.Attributes.RhsaCount)
	assert.Equal(t, 2, output.Data.Attributes.RhbaCount)
	assert.Equal(t, 1, output.Data.Attributes.RheaCount)
	assert.Equal(t, "RHEL", output.Data.Attributes.OSName)
	assert.Equal(t, "8", output.Data.Attributes.OSMajor)
	assert.Equal(t, "10", output.Data.Attributes.OSMinor)
	assert.Equal(t, "8.10", output.Data.Attributes.Rhsm)
	assert.Equal(t, "RHEL 8.10", output.Data.Attributes.OS)
	assert.Equal(t, "baseline_1-1", output.Data.Attributes.BaselineName)
	assert.Equal(t, true, *output.Data.Attributes.BaselineUpToDate)
	assert.Equal(t, int64(1), output.Data.Attributes.BaselineID)
}

func TestSystemDetailDefault2(t *testing.T) {
	core.SetupTest(t)
	// get system with some installable/updatable packages
	w := CreateRequestRouterWithAccount("GET", "/00000000-0000-0000-0000-000000000012", nil, "",
		SystemDetailHandler, "/:inventory_id", 3)

	var output SystemDetailResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 2, output.Data.Attributes.PackagesInstalled)
	assert.Equal(t, 2, output.Data.Attributes.PackagesUpdatable)
}

func TestSystemDetailNoIdProvided(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequest("GET", "/", nil, "", SystemDetailHandler)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, "inventory_id param not found", errResp.Error)
}

func TestSystemDetailNotFound(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/ffffffff-ffff-ffff-ffff-ffffffffffff", nil, "",
		SystemDetailHandler, "/:inventory_id")

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusNotFound, &errResp)
	assert.Equal(t, "inventory not found", errResp.Error)
}

func TestSystemsNoRHSM(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithAccount("GET", "/00000000-0000-0000-0000-000000000014", nil, "",
		SystemDetailHandler, "/:inventory_id", 3)

	var output SystemDetailResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, "00000000-0000-0000-0000-000000000014", output.Data.ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000014", output.Data.Attributes.DisplayName)
	assert.Equal(t, "", output.Data.Attributes.Rhsm)
}

func TestRHSMLessThanOS(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithAccount("GET", "/00000000-0000-0000-0000-000000000003", nil, "",
		SystemDetailHandler, "/:inventory_id", 1)

	var output SystemDetailResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", output.Data.ID)
	assert.Equal(t, "8.0", output.Data.Attributes.Rhsm)
	assert.Equal(t, "8", output.Data.Attributes.OSMajor)
	assert.Equal(t, "1", output.Data.Attributes.OSMinor)
}

func TestRHSMGreaterThanOS(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithAccount("GET", "/00000000-0000-0000-0000-000000000004", nil, "",
		SystemDetailHandler, "/:inventory_id", 1)

	var output SystemDetailResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, "00000000-0000-0000-0000-000000000004", output.Data.ID)
	assert.Equal(t, "8.3", output.Data.Attributes.Rhsm)
	assert.Equal(t, "8", output.Data.Attributes.OSMajor)
	assert.Equal(t, "2", output.Data.Attributes.OSMinor)
}

func TestSystemUnknown(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithAccount("GET", "/unknownsystem", nil, "", SystemDetailHandler, "/:inventory_id", 1)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSystemDetailFiltering(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithAccount("GET", "/00000000-0000-0000-0000-000000000001?filter[filter]=abcd",
		nil, "", SystemDetailHandler, "/:inventory_id", 1)

	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, FilterNotSupportedMsg, errResp.Error)
}
