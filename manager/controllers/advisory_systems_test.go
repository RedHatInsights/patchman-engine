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

func TestAdvisorySystemsDefault(t *testing.T) { //nolint:dupl
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-1", nil)
	initRouterWithPath(AdvisorySystemsListHandler, "/:advisory_id").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output AdvisorySystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 8, len(output.Data))
	assert.Equal(t, "INV-0", output.Data[0].ID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "2018-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.LastUpload.String())
	assert.Equal(t, true, output.Data[0].Attributes.Enabled)
	assert.Equal(t, 2, output.Data[0].Attributes.RhsaCount)
	assert.Equal(t, 3, output.Data[0].Attributes.RhbaCount)
	assert.Equal(t, 3, output.Data[0].Attributes.RheaCount)
	assert.Equal(t, 0, output.Meta.Page)
}

func TestAdvisorySystemsOffsetLimit(t *testing.T) { //nolint:dupl
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-1?offset=5&limit=3", nil)
	initRouterWithPath(AdvisorySystemsListHandler, "/:advisory_id").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output AdvisorySystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "INV-5", output.Data[0].ID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "2018-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.LastUpload.String())
	assert.Equal(t, true, output.Data[0].Attributes.Enabled)
	assert.Equal(t, 0, output.Data[0].Attributes.RhsaCount)
	assert.Equal(t, 1, output.Data[0].Attributes.RheaCount)
	assert.Equal(t, 0, output.Data[0].Attributes.RhbaCount)
	assert.Equal(t, 1, output.Meta.Page)
}

func TestAdvisorySystemsOffsetOverflow(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-1?offset=100&limit=3", nil)
	initRouterWithPath(AdvisorySystemsListHandler, "/:advisory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "too big offset", errResp.Error)
}

func TestAdvisorySystemsSorts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	w := httptest.NewRecorder()

	for _, sort := range SystemsSortFields {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/RH-1?sort=%v", sort), nil)
		initRouterWithPath(AdvisorySystemsListHandler, "/:advisory_id").ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code, "Sort field: ", sort)
	}
}

func TestAdvisorySystemsWrongSort(t *testing.T) { //nolint:dupl
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-1?sort=unknown_key", nil)
	initRouterWithPath(AdvisorySystemsListHandler, "/:advisory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
