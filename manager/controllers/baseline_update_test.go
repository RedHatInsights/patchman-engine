package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"bytes"
	"fmt"
	"net/http"
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
	core.SetupTest(t)
	baselineID := database.CreateBaseline(t, "", testingInventoryIDs, nil)
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

	path := fmt.Sprintf(`/%v`, baselineID)
	w := CreateRequestRouterWithParams("PUT", path, bytes.NewBufferString(data), "", BaselineUpdateHandler, 1,
		"PUT", "/:baseline_id")

	var resp UpdateBaselineResponse
	CheckResponse(t, w, http.StatusOK, &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	desc := "desc"
	database.CheckBaseline(t, baselineID, []string{
		"00000000-0000-0000-0000-000000000004",
		"00000000-0000-0000-0000-000000000006",
		"00000000-0000-0000-0000-000000000007",
		"00000000-0000-0000-0000-000000000008",
	}, `{"to_time": "2022-12-31T12:00:00-04:00"}`, "updated_name", &desc)
	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineWithEmptyAssociations(t *testing.T) {
	core.SetupTest(t)

	baselineID := database.CreateBaseline(t, "", testingInventoryIDs, nil)
	data := `{"inventory_ids": {}}`
	path := fmt.Sprintf(`/%v`, baselineID)
	w := CreateRequestRouterWithParams("PUT", path, bytes.NewBufferString(data), "", BaselineUpdateHandler, 1,
		"PUT", "/:baseline_id")

	var resp UpdateBaselineResponse
	CheckResponse(t, w, http.StatusOK, &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	database.CheckBaseline(t,
		baselineID,
		[]string{
			"00000000-0000-0000-0000-000000000005",
			"00000000-0000-0000-0000-000000000006",
			"00000000-0000-0000-0000-000000000007",
		}, `{"to_time": "2021-01-01T12:00:00-04:00"}`, "temporary_baseline", nil)

	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineShouldRemoveAllAssociations(t *testing.T) {
	core.SetupTest(t)

	baselineID := database.CreateBaseline(t, "", testingInventoryIDs, nil)
	data := `{
		"inventory_ids": {
			"00000000-0000-0000-0000-000000000005": false,
			"00000000-0000-0000-0000-000000000006": false,
			"00000000-0000-0000-0000-000000000007": false
		}
	}`
	path := fmt.Sprintf(`/%v`, baselineID)
	w := CreateRequestRouterWithParams("PUT", path, bytes.NewBufferString(data), "", BaselineUpdateHandler, 1,
		"PUT", "/:baseline_id")

	var resp UpdateBaselineResponse
	CheckResponse(t, w, http.StatusOK, &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	database.CheckBaseline(t, baselineID, []string{},
		`{"to_time": "2021-01-01T12:00:00-04:00"}`, "temporary_baseline", nil)

	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineInvalidPayload(t *testing.T) {
	core.SetupTest(t)

	data := `{"name": 0}`
	w := CreateRequestRouterWithParams("PUT", "/1", bytes.NewBufferString(data), "", BaselineUpdateHandler, 1,
		"PUT", "/:baseline_id")
	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.True(t, strings.Contains(errResp.Error, "name of type string"))
}

func TestUpdateBaselineInvalidSystem(t *testing.T) {
	core.SetupTest(t)

	baselineID := database.CreateBaseline(t, "", testingInventoryIDs, nil)
	data := `{
		"inventory_ids": {
			"00000000-0000-0000-0000-000000000005": false,
			"incorrect_id": false,
			"00000000-0000-0000-0000-000000000009": true
		}
	}`
	path := fmt.Sprintf(`/%v`, baselineID)
	w := CreateRequestRouterWithParams("PUT", path, bytes.NewBufferString(data), "", BaselineUpdateHandler, 1,
		"PUT", "/:baseline_id")

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusNotFound, &errResp)
	assert.Equal(t, "Missing inventory_ids: [00000000-0000-0000-0000-000000000009 incorrect_id]",
		errResp.Error)
	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineNullValues(t *testing.T) {
	core.SetupTest(t)

	baselineID := database.CreateBaseline(t, "", testingInventoryIDs, nil)
	data := `{}`
	path := fmt.Sprintf(`/%v`, baselineID)
	w := CreateRequestRouterWithParams("PUT", path, bytes.NewBufferString(data), "", BaselineUpdateHandler, 1,
		"PUT", "/:baseline_id")

	var resp UpdateBaselineResponse
	CheckResponse(t, w, http.StatusOK, &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	database.CheckBaseline(t, baselineID, testingInventoryIDs, // do not change on null values
		`{"to_time": "2021-01-01T12:00:00-04:00"}`, "temporary_baseline", nil)

	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineInvalidBaselineID(t *testing.T) {
	core.SetupTestEnvironment()
	w := CreateRequestRouterWithParams("PUT", "/invalidBaseline", bytes.NewBufferString("{}"), "", BaselineUpdateHandler,
		1, "PUT", "/:baseline_id")

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, "Invalid baseline_id: invalidBaseline", errResp.Error)
}

func TestUpdateBaselineDuplicatedName(t *testing.T) {
	core.SetupTest(t)
	data := `{"name": "baseline_1-2"}`
	w := CreateRequestRouterWithParams("PUT", "/1", bytes.NewBufferString(data), "", BaselineUpdateHandler, 1,
		"PUT", "/:baseline_id")

	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, DuplicateBaselineNameErr, errResp.Error)
}

func TestUpdateBaselineSystems(t *testing.T) {
	core.SetupTestEnvironment()

	// Assign inventory ID used by baseline 1 to check if it can be reassigned back during update
	baselineID := database.CreateBaseline(t, "test_baseline", []string{"00000000-0000-0000-0000-000000000002"}, nil)

	// Reassigning inventory IDs of another baselines is allowed
	data := `{
		"inventory_ids": {
			"00000000-0000-0000-0000-000000000001": true,
			"00000000-0000-0000-0000-000000000002": true
		}
	}`
	w := CreateRequestRouterWithParams("PUT", "/1", bytes.NewBufferString(data), "", BaselineUpdateHandler, 1,
		"PUT", "/:baseline_id")

	var resp UpdateBaselineResponse
	CheckResponse(t, w, http.StatusOK, &resp)
	assert.Equal(t, int64(1), resp.BaselineID)

	database.DeleteBaseline(t, baselineID)
}

func TestUpdateBaselineSystemsInvalid(t *testing.T) {
	core.SetupTestEnvironment()

	// Removing inventory IDs of another baselines is not allowed
	data := `{
		"inventory_ids": {
			"00000000-0000-0000-0000-000000000001": false,
			"00000000-0000-0000-0000-000000000003": false
		}
	}`
	w := CreateRequestRouterWithParams("PUT", "/1", bytes.NewBufferString(data), "", BaselineUpdateHandler, 1,
		"PUT", "/:baseline_id")

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, "Invalid inventory IDs: unable to update systems of another baseline", errResp.Error)
}

func TestUpdateBaselineEmptyDescription(t *testing.T) {
	core.SetupTestEnvironment()

	desc := "empty_update_description"
	baselineID := database.CreateBaseline(t, desc, testingInventoryIDs, &desc)
	defer database.DeleteBaseline(t, baselineID)

	data := `{"description": ""}`
	w := CreateRequestRouterWithParams("PUT", fmt.Sprintf(`/%v`, baselineID), bytes.NewBufferString(data), "",
		BaselineUpdateHandler, 1, "PUT", "/:baseline_id")

	var resp UpdateBaselineResponse
	CheckResponse(t, w, http.StatusOK, &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	database.CheckBaseline(t, resp.BaselineID, testingInventoryIDs, `{"to_time": "2021-01-01T12:00:00-04:00"}`, desc, nil)
}

func TestUpdateBaselineNilDescription(t *testing.T) {
	core.SetupTestEnvironment()

	desc := "nil_update_description"
	baselineID := database.CreateBaseline(t, desc, testingInventoryIDs, &desc)
	defer database.DeleteBaseline(t, baselineID)

	data := `{"name": "new_name", "description": null}`
	w := CreateRequestRouterWithParams("PUT", fmt.Sprintf(`/%v`, baselineID), bytes.NewBufferString(data), "",
		BaselineUpdateHandler, 1, "PUT", "/:baseline_id")

	var resp UpdateBaselineResponse
	CheckResponse(t, w, http.StatusOK, &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	database.CheckBaseline(t, resp.BaselineID, testingInventoryIDs,
		`{"to_time": "2021-01-01T12:00:00-04:00"}`, "new_name", &desc)

	// missing description
	data = `{"name": "new_name"}`
	w = CreateRequestRouterWithParams("PUT", fmt.Sprintf(`/%v`, baselineID), bytes.NewBufferString(data), "",
		BaselineUpdateHandler, 1, "PUT", "/:baseline_id")

	CheckResponse(t, w, http.StatusOK, &resp)
	assert.Equal(t, baselineID, resp.BaselineID)
	database.CheckBaseline(t, resp.BaselineID, testingInventoryIDs,
		`{"to_time": "2021-01-01T12:00:00-04:00"}`, "new_name", &desc)
}

func TestUpdateBaselineSpacesDescription(t *testing.T) {
	core.SetupTestEnvironment()

	desc := "spaces_update_description"
	baselineID := database.CreateBaseline(t, desc, testingInventoryIDs, &desc)
	defer database.DeleteBaseline(t, baselineID)

	data := `{"description": "   "}`
	w := CreateRequestRouterWithParams("PUT", fmt.Sprintf(`/%v`, baselineID), bytes.NewBufferString(data), "",
		BaselineUpdateHandler, 1, "PUT", "/:baseline_id")

	var err utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &err)
}
