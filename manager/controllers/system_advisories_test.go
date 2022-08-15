package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemAdvisoriesDefault(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/00000000-0000-0000-0000-000000000001", nil, "",
		SystemAdvisoriesHandler, "/:inventory_id")

	var output SystemAdvisoriesResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 8, len(output.Data))
	assert.Equal(t, "RH-7", output.Data[0].ID)
	assert.Equal(t, "advisory", output.Data[0].Type)
	assert.Equal(t, "adv-7-des", output.Data[0].Attributes.Description)
	assert.Equal(t, "adv-7-syn", output.Data[0].Attributes.Synopsis)
	assert.Equal(t, 1, output.Data[0].Attributes.AdvisoryType)
	assert.Equal(t, "enhancement", output.Data[0].Attributes.AdvisoryTypeName)
	assert.Equal(t, "2017-09-22 19:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, "2017-09-22 19:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, 0, output.Data[0].Attributes.CveCount)
	assert.Equal(t, false, output.Data[0].Attributes.RebootRequired)
}

func TestSystemAdvisoriesIDsDefault(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/00000000-0000-0000-0000-000000000001", nil, "",
		SystemAdvisoriesIDsHandler, "/:inventory_id")

	var output IDsResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 8, len(output.IDs))
	assert.Equal(t, "RH-7", output.IDs[0])
}

func TestSystemAdvisoriesNotFound(t *testing.T) { //nolint:dupl
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/nonexistant/advisories", nil, "",
		SystemAdvisoriesHandler, "/:inventory_id")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSystemAdvisoriesIDsNotFound(t *testing.T) { //nolint:dupl
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/nonexistant/advisories", nil, "",
		SystemAdvisoriesIDsHandler, "/:inventory_id")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSystemAdvisoriesOffsetLimit(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/00000000-0000-0000-0000-000000000001?offset=4&limit=3", nil, "",
		SystemAdvisoriesHandler, "/:inventory_id")

	var output SystemAdvisoriesResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "RH-1", output.Data[0].ID)
}

func TestSystemAdvisoriesIDsOffsetLimit(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/00000000-0000-0000-0000-000000000001?offset=4&limit=3", nil, "",
		SystemAdvisoriesIDsHandler, "/:inventory_id")

	var output IDsResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 3, len(output.IDs))
	assert.Equal(t, "RH-1", output.IDs[0])
}

func TestSystemAdvisoriesOffsetOverflow(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/00000000-0000-0000-0000-000000000001?offset=100&limit=3", nil, "",
		SystemAdvisoriesHandler, "/:inventory_id")

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestSystemAdvisoriesPossibleSorts(t *testing.T) {
	core.SetupTest(t)

	for sort := range SystemAdvisoriesFields {
		if sort == "ReleaseVersions" {
			// this field is not sortable, skip it
			continue
		}
		w := CreateRequestRouterWithPath("GET", fmt.Sprintf("/00000000-0000-0000-0000-000000000001?sort=%v", sort),
			nil, "", SystemAdvisoriesHandler, "/:inventory_id")

		var output SystemAdvisoriesResponse
		CheckResponse(t, w, http.StatusOK, &output)
		assert.Equal(t, output.Meta.Sort[0], sort)
	}
}

func TestSystemAdvisoriesWrongSort(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/00000000-0000-0000-0000-000000000001?sort=unknown_key", nil, "",
		SystemAdvisoriesHandler, "/:inventory_id")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSystemAdvisoriesSearch(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/00000000-0000-0000-0000-000000000001?search=h-3", nil, "",
		SystemAdvisoriesHandler, "/:inventory_id")

	var output SystemAdvisoriesResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "RH-3", output.Data[0].ID)
	assert.Equal(t, "advisory", output.Data[0].Type)
	assert.Equal(t, "adv-3-des", output.Data[0].Attributes.Description)
	assert.Equal(t, "adv-3-syn", output.Data[0].Attributes.Synopsis)
	assert.Equal(t, 2, output.Data[0].Attributes.CveCount)
	assert.Equal(t, false, output.Data[0].Attributes.RebootRequired)
}

func TestSystemAdvisoriesWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/:inventory_id", "/00000000-0000-0000-0000-000000000001?offset=1000",
		SystemAdvisoriesHandler)
}

func TestSystemAdvisoriesExportUnknown(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/unknownsystem", nil, "", SystemAdvisoriesHandler, "/:inventory_id")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
