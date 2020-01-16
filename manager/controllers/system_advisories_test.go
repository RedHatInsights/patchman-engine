package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSystemAdvisoriesDefault(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/INV-0", nil)
	initRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemAdvisoriesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 8, len(output.Data))
	assert.Equal(t, "RH-1", output.Data[0].ID)
	assert.Equal(t, "advisory", output.Data[0].Type)
	assert.Equal(t, "adv-1-des", output.Data[0].Attributes.Description)
	assert.Equal(t, "adv-1-syn", output.Data[0].Attributes.Synopsis)
	assert.Equal(t, 1, output.Data[0].Attributes.AdvisoryType)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, 0, output.Data[0].Attributes.ApplicableSystems)
	assert.Equal(t, 0, output.Meta.Page)
}

func TestSystemAdvisoriesOffsetLimit(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/INV-0?offset=4&limit=3", nil)
	initRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemAdvisoriesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "RH-5", output.Data[0].ID)
	assert.Equal(t, 0, output.Data[0].Attributes.ApplicableSystems)
	assert.Equal(t, 1, output.Meta.Page)
}

func TestSystemAdvisoriesOffsetOverflow(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/INV-0?offset=100&limit=3", nil)
	initRouterWithPath(SystemAdvisoriesHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "too big offset", errResp.Error)
}
