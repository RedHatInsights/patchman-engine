package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdvisoryDetailDefault(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-9", nil)
	core.InitRouterWithPath(AdvisoryDetailHandler, "/:advisory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output AdvisoryDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	// data
	assert.Equal(t, "advisory", output.Data.Type)
	assert.Equal(t, "RH-9", output.Data.ID)
	assert.Equal(t, "adv-9-syn", output.Data.Attributes.Synopsis)
	assert.Equal(t, "adv-9-des", output.Data.Attributes.Description)
	assert.Equal(t, "adv-9-sol", output.Data.Attributes.Solution)
	assert.Equal(t, "2016-09-22 20:00:00 +0000 UTC", output.Data.Attributes.PublicDate.String())
	assert.Equal(t, "2018-09-22 20:00:00 +0000 UTC", output.Data.Attributes.ModifiedDate.String())
	assert.Equal(t, 1, len(output.Data.Attributes.Packages))
	assert.Equal(t, "77.0.1-1.fc31.x86_64", output.Data.Attributes.Packages["firefox"])
	assert.Nil(t, output.Data.Attributes.Severity)
}

func TestAdvisoryDetailCVE(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-3", nil)
	core.InitRouterWithPath(AdvisoryDetailHandler, "/:advisory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output AdvisoryDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output.Data.Attributes.Cves))
	assert.Equal(t, "CVE-1", output.Data.Attributes.Cves[0])
	assert.Equal(t, "CVE-2", output.Data.Attributes.Cves[1])
}

func TestAdvisoryNoIdProvided(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	core.InitRouter(AdvisoryDetailHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "advisory_id param not found", errResp.Error)
}

func TestAdvisoryNotFound(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	core.InitRouterWithPath(AdvisoryDetailHandler, "/:advisory_id").ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "advisory not found", errResp.Error)
}
