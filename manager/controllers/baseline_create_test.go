package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateBaseline(t *testing.T) {
	core.SetupTest(t)
	data := `{
		"name": "my_baseline",
		"inventory_ids": [
			"00000000-0000-0000-0000-000000000005",
			"00000000-0000-0000-0000-000000000006"
		],
        "config": {"to_time": "2022-12-31T12:00:00-04:00"},
		"description": "desc"
	}`
	w := CreateRequestRouterWithParams("PUT", "/", "", "", bytes.NewBufferString(data), "", CreateBaselineHandler, 1)

	var resp CreateBaselineResponse
	ParseResponseBody(t, w.Body.Bytes(), &resp)
	desc := "desc"
	database.CheckBaseline(t, resp.BaselineID, []string{
		"00000000-0000-0000-0000-000000000005",
		"00000000-0000-0000-0000-000000000006",
	}, `{"to_time": "2022-12-31T12:00:00-04:00"}`, "my_baseline", &desc)
	database.DeleteBaseline(t, resp.BaselineID)
}

func TestCreateBaselineNameOnly(t *testing.T) {
	core.SetupTest(t)
	data := `{"name": "my_empty_baseline"}`
	w := CreateRequestRouterWithParams("PUT", "/", "", "", bytes.NewBufferString(data), "", CreateBaselineHandler, 1)

	var resp CreateBaselineResponse
	ParseResponseBody(t, w.Body.Bytes(), &resp)
	database.CheckBaseline(t, resp.BaselineID, []string{}, "", "my_empty_baseline", nil)
	database.DeleteBaseline(t, resp.BaselineID)
}

func TestCreateBaselineNameEmptyString(t *testing.T) {
	core.SetupTest(t)
	data := `{"name": ""}`
	w := CreateRequestRouterWithParams("PUT", "/", "", "", bytes.NewBufferString(data), "", CreateBaselineHandler, 1)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, BaselineMissingNameErr, errResp.Error)
}

func TestCreateBaselineMissingName(t *testing.T) {
	core.SetupTest(t)
	data := `{}`
	w := CreateRequestRouterWithParams("PUT", "/", "", "", bytes.NewBufferString(data), "", CreateBaselineHandler, 1)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, BaselineMissingNameErr, errResp.Error)
}

func TestCreateBaselineInvalidRequest(t *testing.T) {
	core.SetupTest(t)
	data := `{"name": 0}`
	w := CreateRequestRouterWithParams("PUT", "/", "", "", bytes.NewBufferString(data), "", CreateBaselineHandler, 1)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.True(t, strings.Contains(errResp.Error,
		"cannot unmarshal number into Go struct field CreateBaselineRequest.name of type string"))
}

func TestCreateBaselineDuplicatedName(t *testing.T) {
	core.SetupTest(t)
	data := `{
		"name": "baseline_1-1"
	}`
	w := CreateRequestRouterWithParams("PUT", "/", "", "", bytes.NewBufferString(data), "", CreateBaselineHandler, 1)

	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, DuplicateBaselineNameErr, errResp.Error)
}

func TestCreateBaselineDescriptionEmptyString(t *testing.T) {
	core.SetupTest(t)
	data := `{"name": "baseline_empty_desc", "description": ""}`
	w := CreateRequestRouterWithParams("PUT", "/", "", "", bytes.NewBufferString(data), "", CreateBaselineHandler, 1)

	var resp CreateBaselineResponse
	CheckResponse(t, w, http.StatusOK, &resp)
	database.CheckBaseline(t, resp.BaselineID, []string{}, "", "baseline_empty_desc", nil)
	database.DeleteBaseline(t, resp.BaselineID)
}

func TestCreateBaselineDescriptionSpaces(t *testing.T) {
	core.SetupTest(t)
	data := `{"name": "baseline_spaces_desc", "description": "   "}`
	w := CreateRequestRouterWithParams("PUT", "/", "", "", bytes.NewBufferString(data), "", CreateBaselineHandler, 1)

	var err utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &err)
}

func TestCreateBaselineSatelliteSystem(t *testing.T) {
	core.SetupTest(t)

	system := models.SystemPlatform{
		InventoryID:      "99999999-0000-0000-0000-000000000015",
		DisplayName:      "satellite_system_test",
		RhAccountID:      1,
		BuiltPkgcache:    true,
		SatelliteManaged: true,
	}
	database.CreateSystem(t, &system)
	defer database.DeleteSystem(t, system.InventoryID)

	data := `{
		"name": "baseline_satellite",
		"inventory_ids": [
			"00000000-0000-0000-0000-000000000005",
			"99999999-0000-0000-0000-000000000015"
		],
        "config": {"to_time": "2022-12-31T12:00:00-04:00"},
		"description": "desc"
	}`

	w := CreateRequestRouterWithParams("PUT", "/", "", "", bytes.NewBufferString(data), "", CreateBaselineHandler, 1)
	var err utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &err)
}
