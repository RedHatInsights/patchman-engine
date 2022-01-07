package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemAdvisoriesDefault(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000001", nil)
	core.InitRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemAdvisoriesResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)
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

func TestSystemAdvisoriesNotFound(t *testing.T) { //nolint:dupl
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nonexistant/advisories", nil)
	core.InitRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSystemAdvisoriesOffsetLimit(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000001?offset=4&limit=3", nil)
	core.InitRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemAdvisoriesResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "RH-1", output.Data[0].ID)
}

func TestSystemAdvisoriesOffsetOverflow(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000001?offset=100&limit=3", nil)
	core.InitRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestSystemAdvisoriesPossibleSorts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	for sort := range SystemAdvisoriesFields {
		if sort == "ReleaseVersions" {
			// this fiesd is not sortable, skip it
			continue
		}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/00000000-0000-0000-0000-000000000001?sort=%v", sort), nil)
		core.InitRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

		var output SystemAdvisoriesResponse
		ParseResponseBody(t, w.Body.Bytes(), &output)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, output.Meta.Sort[0], sort)
	}
}

func TestSystemAdvisoriesWrongSort(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000001?sort=unknown_key", nil)
	core.InitRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSystemAdvisoriesSearch(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000001?search=h-3", nil)
	core.InitRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemAdvisoriesResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)
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
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unknownsystem", nil)
	core.InitRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
