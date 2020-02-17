package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSystemsDefault(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	initRouter(SystemsListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	// data
	assert.Equal(t, 8, len(output.Data))
	assert.Equal(t, "INV-0", output.Data[0].ID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "2018-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.LastUpload.String())
	assert.Equal(t, "2018-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.LastEvaluation.String())
	assert.Equal(t, 3, output.Data[0].Attributes.RheaCount)
	assert.Equal(t, 3, output.Data[0].Attributes.RhbaCount)
	assert.Equal(t, 2, output.Data[0].Attributes.RhsaCount)
	assert.Equal(t, true, output.Data[0].Attributes.Enabled)

	// links
	assert.Equal(t, "/api/patch/v1/systems?offset=0&limit=25&sort=id", output.Links.First)
	assert.Equal(t, "/api/patch/v1/systems?offset=0&limit=25&sort=id", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// test meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 8, output.Meta.TotalItems)
}

func TestSystemsOffsetLimit(t *testing.T) { //nolint:dupl
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=0&limit=4", nil)
	initRouter(SystemsListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 8, output.Meta.TotalItems)
}

func TestSystemsOffset(t *testing.T) { //nolint:dupl
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=4&limit=4", nil)
	initRouter(SystemsListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 4, output.Meta.Offset)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 8, output.Meta.TotalItems)
}

func TestSystemsOffsetOverflow(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=13&limit=4", nil)
	initRouter(SystemsListHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestSystemsWrongSort(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?sort=unknown_key", nil)
	initRouter(SystemsListHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
