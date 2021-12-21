package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBaselineDelete(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	var inventoryIDs = []string{
		"00000000-0000-0000-0000-000000000005",
		"00000000-0000-0000-0000-000000000006",
		"00000000-0000-0000-0000-000000000007"}
	baselineID := database.CreateBaseline(t, inventoryIDs)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf(`/%v`, baselineID), nil)
	core.InitRouterWithPath(BaselineDeleteHandler, "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	database.CheckBaselineDeleted(t, baselineID)
}

func TestBaselineDeleteNonExisting(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/88888", nil)
	core.InitRouterWithPath(BaselineDeleteHandler, "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBaselineDeleteInvalid(t *testing.T) {
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/invalidBaseline", nil)
	core.InitRouterWithPath(BaselineDeleteHandler, "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, errResp.Error, "Invalid baseline_id: invalidBaseline")
}
