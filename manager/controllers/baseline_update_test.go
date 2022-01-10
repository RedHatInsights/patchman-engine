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
		"config": {
			"to_time": "2021-01-01 12:00:00-04"
		}
	}`

	w := httptest.NewRecorder()
	path := fmt.Sprintf(`/%v`, baselineID)
	req, _ := http.NewRequest("POST", path, bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "POST", "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	database.CheckBaseline(t, baselineID, []string{
		"00000000-0000-0000-0000-000000000004",
		"00000000-0000-0000-0000-000000000006",
		"00000000-0000-0000-0000-000000000007",
		"00000000-0000-0000-0000-000000000008",
	},
		`{"to_time": "2021-01-01 12:00:00-04"}`,
		"updated_name")
	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineWithEmptyAssociations(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	baselineID := database.CreateBaseline(t, testingInventoryIDs)

	data := `{
		"name": "updated_name",
		"inventory_ids": {},
		"config": {
			"to_time": "2021-01-01 12:00:00-04"
		}
	}`

	w := httptest.NewRecorder()
	path := fmt.Sprintf(`/%v`, baselineID)
	req, _ := http.NewRequest("POST", path, bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "POST", "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	database.CheckBaseline(t,
		baselineID,
		[]string{
			"00000000-0000-0000-0000-000000000005",
			"00000000-0000-0000-0000-000000000006",
			"00000000-0000-0000-0000-000000000007",
		},
		`{"to_time": "2021-01-01 12:00:00-04"}`,
		"updated_name")

	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineShouldRemoveAllAssociations(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	baselineID := database.CreateBaseline(t, testingInventoryIDs)

	data := `{
		"name": "updated_name",
		"inventory_ids": {
			"00000000-0000-0000-0000-000000000005": false,
			"00000000-0000-0000-0000-000000000006": false,
			"00000000-0000-0000-0000-000000000007": false
		},
		"config": {
			"to_time": "2021-01-01 12:00:00-04"
		}
	}`

	w := httptest.NewRecorder()
	path := fmt.Sprintf(`/%v`, baselineID)
	req, _ := http.NewRequest("POST", path, bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "POST", "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	database.CheckBaseline(t,
		baselineID,
		[]string{},
		`{"to_time": "2021-01-01 12:00:00-04"}`,
		"updated_name")

	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineInvalidPayload(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	data := `{
		"name": 0
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/1", bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "POST", "/:baseline_id").ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.True(t, strings.Contains(errResp.Error, "name of type string"))
}

func TestUpdateBaselineInvalidSystem(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	baselineID := database.CreateBaseline(t, testingInventoryIDs)
	data := `{
		"name": "updated_name",
		"inventory_ids": {
			"00000000-0000-0000-0000-000000000005": false,
			"incorrect_id": false,
			"00000000-0000-0000-0000-000000000009": true
		},
		"config": {
			"to_time": "2021-01-01 12:00:00-04"
		}
	}`

	w := httptest.NewRecorder()
	path := fmt.Sprintf(`/%v`, baselineID)
	req, _ := http.NewRequest("POST", path, bytes.NewBufferString(data))
	core.InitRouterWithParams(BaselineUpdateHandler, 1, "POST", "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp.Error)
	assert.True(t, strings.Contains(errResp.Error, "System(s) do(es) not exist"))

	database.DeleteBaseline(t, baselineID)
}
