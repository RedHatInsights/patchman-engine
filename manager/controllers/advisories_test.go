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

func TestAdvisoriesDefault(t *testing.T) { //nolint:dupl
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	initRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output AdvisoriesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	// data
	assert.Equal(t, 8, len(output.Data))
	assert.Equal(t, "RH-1", output.Data[0].ID)
	assert.Equal(t, "advisory", output.Data[0].Type)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, "adv-1-des", output.Data[0].Attributes.Description)
	assert.Equal(t, "adv-1-syn", output.Data[0].Attributes.Synopsis)
	assert.Equal(t, 7, output.Data[0].Attributes.ApplicableSystems)

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
	assert.Equal(t, 8, output.Meta.TotalItems)
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
	assert.Equal(t, 8, output.Meta.TotalItems)
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
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 1, output.Meta.Offset)
	assert.Equal(t, 0, output.Meta.Page)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 4, output.Meta.PageSize)
	assert.Equal(t, 8, output.Meta.TotalItems)
}

func TestAdvisoriesOffsetOverflow(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=13&limit=4", nil)
	initRouter(AdvisoriesListHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "too big offset", errResp.Error)
}

func TestAdvisoriesOrder(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?sort=-public_date", nil)
	initRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output AdvisoriesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	// Advisoiry RH-7 has latest public date
	assert.Equal(t, 8, len(output.Data))
	assert.Equal(t, "RH-7", output.Data[0].ID)
	assert.Equal(t, "advisory", output.Data[0].Type)
	assert.Equal(t, "2017-09-22 19:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, "adv-7-des", output.Data[0].Attributes.Description)
	assert.Equal(t, "adv-7-syn", output.Data[0].Attributes.Synopsis)
	assert.Equal(t, 1, output.Data[0].Attributes.ApplicableSystems)
}

func TestAdvisoriesPossibleSorts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	w := httptest.NewRecorder()

	for _, sort := range AdvisoriesSortFields {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/?sort=%v", sort), nil)
		initRouter(AdvisoriesListHandler).ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code, "Sort field: ", sort)
	}
}

func TestAdvisoriesWrongSort(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?sort=unknown_key", nil)
	initRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdvisoriesSearch(t *testing.T) { //nolint:dupl
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?search=h-3", nil)
	initRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output AdvisoriesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	// data
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "RH-3", output.Data[0].ID)
	assert.Equal(t, "advisory", output.Data[0].Type)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, "adv-3-des", output.Data[0].Attributes.Description)
	assert.Equal(t, "adv-3-syn", output.Data[0].Attributes.Synopsis)
	assert.Equal(t, 1, output.Data[0].Attributes.ApplicableSystems)

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
	assert.Equal(t, 1, output.Meta.TotalItems)
}
