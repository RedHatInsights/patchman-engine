package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSystemDetailDefault(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/INV-0", nil)
	initRouterWithPath(SystemDetailHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, "INV-0", output.Data.Id)
	assert.Equal(t, "system", output.Data.Type)
	assert.Nil(t, output.Data.Attributes.LastEvaluation)
	assert.Equal(t, "2018-09-22 16:00:00 +0000 UTC", output.Data.Attributes.LastUpload.String())
	assert.Equal(t, 8, output.Data.Attributes.RhsaCount)
}

func TestSystemDetailNoIdProvided(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	initRouter(SystemDetailHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "inventory_id param not found", errResp.Error)
}

func TestSystemDetailNotFound(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	initRouterWithPath(SystemDetailHandler, "/:inventory_id").ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "inventory not found", errResp.Error)
}
