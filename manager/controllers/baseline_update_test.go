package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testingInventoryIDs = []string{
	"00000000-0000-0000-0000-000000000005",
	"00000000-0000-0000-0000-000000000006",
	"00000000-0000-0000-0000-000000000007",
}

func TestUpdateBaseline(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	baselineID := database.CreateBaseline(t, testingInventoryIDs)
	data := `{
		"name": "updated_name",
		"inventory_ids": {
			"00000000-0000-0000-0000-000000000004": true,
			"00000000-0000-0000-0000-000000000005": false,
			"00000000-0000-0000-0000-000000000008": true
		},
        "config": {"to_time": "2022-12-31T12:00:00-04:00"},
		"description": "desc"
	}`
	w := httptest.NewRecorder()
	path := fmt.Sprintf(`/%v`, baselineID)
	req, _ := http.NewRequest("PUT", path, bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "PUT", "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp UpdateBaselineResponse
	ParseResponseBody(t, w.Body.Bytes(), &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	database.CheckBaseline(t, baselineID, []string{
		"00000000-0000-0000-0000-000000000004",
		"00000000-0000-0000-0000-000000000006",
		"00000000-0000-0000-0000-000000000007",
		"00000000-0000-0000-0000-000000000008",
	}, `{"to_time": "2022-12-31T12:00:00-04:00"}`, "updated_name", "desc")
	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineWithEmptyAssociations(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	baselineID := database.CreateBaseline(t, testingInventoryIDs)
	data := `{"inventory_ids": {}}`
	w := httptest.NewRecorder()
	path := fmt.Sprintf(`/%v`, baselineID)
	req, _ := http.NewRequest("PUT", path, bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "PUT", "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp UpdateBaselineResponse
	ParseResponseBody(t, w.Body.Bytes(), &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	database.CheckBaseline(t,
		baselineID,
		[]string{
			"00000000-0000-0000-0000-000000000005",
			"00000000-0000-0000-0000-000000000006",
			"00000000-0000-0000-0000-000000000007",
		}, `{"to_time": "2021-01-01T12:00:00-04:00"}`, "temporary_baseline", "")

	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineShouldRemoveAllAssociations(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	baselineID := database.CreateBaseline(t, testingInventoryIDs)
	data := `{
		"inventory_ids": {
			"00000000-0000-0000-0000-000000000005": false,
			"00000000-0000-0000-0000-000000000006": false,
			"00000000-0000-0000-0000-000000000007": false
		}
	}`
	w := httptest.NewRecorder()
	path := fmt.Sprintf(`/%v`, baselineID)
	req, _ := http.NewRequest("PUT", path, bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "PUT", "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp UpdateBaselineResponse
	ParseResponseBody(t, w.Body.Bytes(), &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	database.CheckBaseline(t, baselineID, []string{},
		`{"to_time": "2021-01-01T12:00:00-04:00"}`, "temporary_baseline", "")

	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineInvalidPayload(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	data := `{"name": 0}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/1", bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "PUT", "/:baseline_id").ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.True(t, strings.Contains(errResp.Error, "name of type string"))
}

func TestUpdateBaselineInvalidSystem(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	baselineID := database.CreateBaseline(t, testingInventoryIDs)
	data := `{
		"inventory_ids": {
			"00000000-0000-0000-0000-000000000005": false,
			"incorrect_id": false,
			"00000000-0000-0000-0000-000000000009": true
		}
	}`
	w := httptest.NewRecorder()
	path := fmt.Sprintf(`/%v`, baselineID)
	req, _ := http.NewRequest("PUT", path, bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "PUT", "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "Missing inventory_ids: [00000000-0000-0000-0000-000000000009 incorrect_id]",
		errResp.Error)
	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineNullValues(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	baselineID := database.CreateBaseline(t, testingInventoryIDs)
	data := `{}`
	w := httptest.NewRecorder()
	path := fmt.Sprintf(`/%v`, baselineID)
	req, _ := http.NewRequest("PUT", path, bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "PUT", "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp UpdateBaselineResponse
	ParseResponseBody(t, w.Body.Bytes(), &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	database.CheckBaseline(t, baselineID, testingInventoryIDs, // do not change on null values
		`{"to_time": "2021-01-01T12:00:00-04:00"}`, "temporary_baseline", "")

	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineInvalidBaselineID(t *testing.T) {
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/invalidBaseline", bytes.NewBufferString("{}"))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "PUT", "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "Invalid baseline_id: invalidBaseline", errResp.Error)
}

func TestUpdateBaselineDuplicatedName(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	data := `{"name": "baseline_1-2"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/1", bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "PUT", "/:baseline_id").ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "baseline name already exists", errResp.Error)
}
