package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSystemDetailDefault1(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000001", nil)
	core.InitRouterWithPath(SystemDetailHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data.ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data.Attributes.DisplayName)
	assert.Equal(t, "system", output.Data.Type)
	assert.Equal(t, "2018-09-22 16:00:00 +0000 UTC", output.Data.Attributes.LastEvaluation.String())
	assert.Equal(t, "2020-09-22 16:00:00 +0000 UTC", output.Data.Attributes.LastUpload.String())
	assert.False(t, output.Data.Attributes.Stale)
	assert.True(t, output.Data.Attributes.ThirdParty)
	assert.Equal(t, 2, output.Data.Attributes.RhsaCount)
	assert.Equal(t, 3, output.Data.Attributes.RhbaCount)
	assert.Equal(t, 3, output.Data.Attributes.RheaCount)
	assert.Equal(t, "RHEL", output.Data.Attributes.OSName)
	assert.Equal(t, "8", output.Data.Attributes.OSMajor)
	assert.Equal(t, "1", output.Data.Attributes.OSMinor)
	assert.Equal(t, "8.1", output.Data.Attributes.Rhsm)
}

func TestSystemDetailDefault2(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	// get system with some installable/updatable packages
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000012", nil)
	core.InitRouterWithAccount(SystemDetailHandler, "/:inventory_id", 3).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, output.Data.Attributes.PackagesInstalled)
	assert.Equal(t, 2, output.Data.Attributes.PackagesUpdatable)
}

func TestSystemDetailNoIdProvided(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	core.InitRouter(SystemDetailHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "inventory_id param not found", errResp.Error)
}

func TestSystemDetailNotFound(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ffffffff-ffff-ffff-ffff-ffffffffffff", nil)
	core.InitRouterWithPath(SystemDetailHandler, "/:inventory_id").ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "inventory not found", errResp.Error)
}

func TestSystemsNoRHSM(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000014", nil)
	core.InitRouterWithAccount(SystemDetailHandler, "/:inventory_id", 3).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, "00000000-0000-0000-0000-000000000014", output.Data.ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000014", output.Data.Attributes.DisplayName)
	assert.Equal(t, "", output.Data.Attributes.Rhsm)
}

func TestRHSMLessThanOS(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000003", nil)
	core.InitRouterWithAccount(SystemDetailHandler, "/:inventory_id", 1).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", output.Data.ID)
	assert.Equal(t, "8.0", output.Data.Attributes.Rhsm)
	assert.Equal(t, "8", output.Data.Attributes.OSMajor)
	assert.Equal(t, "1", output.Data.Attributes.OSMinor)
}

func TestRHSMGreaterThanOS(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000004", nil)
	core.InitRouterWithAccount(SystemDetailHandler, "/:inventory_id", 1).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output SystemDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, "00000000-0000-0000-0000-000000000004", output.Data.ID)
	assert.Equal(t, "8.3", output.Data.Attributes.Rhsm)
	assert.Equal(t, "8", output.Data.Attributes.OSMajor)
	assert.Equal(t, "2", output.Data.Attributes.OSMinor)
}

func TestSystemUnknown(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unknownsystem", nil)
	core.InitRouterWithAccount(SystemDetailHandler, "/:inventory_id", 1).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
