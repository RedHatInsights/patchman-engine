package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdvisoriesDefault(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	initRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output AdvisoriesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	// data
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "RH-1", output.Data[0].Id)
	assert.Equal(t, "advisory", output.Data[0].Type)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, "adv-1-des", output.Data[0].Attributes.Description)
	assert.Equal(t, "adv-1-syn", output.Data[0].Attributes.Synopsis)

	// links
	assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=25&data_format=json", output.Links.First)
	assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=25&data_format=json", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 0, output.Meta.Page)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, core.DefaultLimit, output.Meta.PageSize)
	assert.Equal(t, 3, output.Meta.TotalItems)
}

func TestAdvisoriesOffsetLimit(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=0&limit=2", nil)
	initRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output AdvisoriesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 0, output.Meta.Page)
	assert.Equal(t, 2, output.Meta.Limit)
	assert.Equal(t, 2, output.Meta.PageSize)
	assert.Equal(t, 3, output.Meta.TotalItems)
}

func TestAdvisoriesOffset(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=1&limit=4", nil)
	initRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output AdvisoriesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, 1, output.Meta.Offset)
	assert.Equal(t, 0, output.Meta.Page)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 4, output.Meta.PageSize)
	assert.Equal(t, 3, output.Meta.TotalItems)
}

func TestAdvisoriesOffsetOverflow(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=13&limit=4", nil)
	initRouter(AdvisoriesListHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "too big offset", errResp.Error)
}
