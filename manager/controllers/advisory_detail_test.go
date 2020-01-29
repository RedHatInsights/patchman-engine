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
	req, _ := http.NewRequest("GET", "/RH-1", nil)
	initRouterWithPath(AdvisoryDetailHandler, "/:advisory_id").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output AdvisoryDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	// data
	assert.Equal(t, "advisory", output.Data.Type)
	assert.Equal(t, "RH-1", output.Data.ID)
	assert.Equal(t, "adv-1-syn", output.Data.Attributes.Synopsis)
	assert.Equal(t, "adv-1-des", output.Data.Attributes.Description)
	assert.Equal(t, "adv-1-sol", output.Data.Attributes.Solution)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", output.Data.Attributes.PublicDate.String())
	assert.Equal(t, "2017-09-22 16:00:00 +0000 UTC", output.Data.Attributes.ModifiedDate.String())
	assert.Nil(t, output.Data.Attributes.Severity)
}

func TestAdvisoryNoIdProvided(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	initRouter(AdvisoryDetailHandler).ServeHTTP(w, req)
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
	initRouterWithPath(AdvisoryDetailHandler, "/:advisory_id").ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "advisory not found", errResp.Error)
}
