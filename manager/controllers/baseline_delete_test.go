package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaselineDelete(t *testing.T) {
	core.SetupTest(t)
	var inventoryIDs = []string{
		"00000000-0000-0000-0000-000000000005",
		"00000000-0000-0000-0000-000000000006",
		"00000000-0000-0000-0000-000000000007"}
	baselineID := database.CreateBaseline(t, "", inventoryIDs)

	w := CreateRequestRouterWithPath("GET", fmt.Sprintf(`/%v`, baselineID), nil, nil, BaselineDeleteHandler,
		"/:baseline_id")

	var resp DeleteBaselineResponse
	ParseResponse(t, w, http.StatusOK, &resp)
	database.CheckBaselineDeleted(t, resp.BaselineID)
}

func TestBaselineDeleteNonExisting(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/88888", nil, nil, BaselineDeleteHandler, "/:baseline_id")

	var errResp utils.ErrorResponse
	ParseResponse(t, w, http.StatusNotFound, &errResp)
	assert.Equal(t, "baseline not found", errResp.Error)
}

func TestBaselineDeleteInvalid(t *testing.T) {
	core.SetupTestEnvironment()
	w := CreateRequestRouterWithPath("GET", "/invalidBaseline", nil, nil, BaselineDeleteHandler, "/:baseline_id")

	var errResp utils.ErrorResponse
	ParseResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, "Invalid baseline_id: invalidBaseline", errResp.Error)
}
