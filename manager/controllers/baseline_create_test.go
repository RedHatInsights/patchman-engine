package controllers

import (
	"app/base/core"
	"app/base/database"
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
	w := CreateRequestRouterWithParams("PUT", "/", bytes.NewBufferString(data), nil, CreateBaselineHandler, 1, "PUT", "/")

	assert.Equal(t, http.StatusOK, w.Code)
	var resp CreateBaselineResponse
	ParseResponseBody(t, w.Body.Bytes(), &resp)
	database.CheckBaseline(t, resp.BaselineID, []string{
		"00000000-0000-0000-0000-000000000005",
		"00000000-0000-0000-0000-000000000006",
	}, `{"to_time": "2022-12-31T12:00:00-04:00"}`, "my_baseline", "desc")
	database.DeleteBaseline(t, resp.BaselineID)
}

func TestCreateBaselineNameOnly(t *testing.T) {
	core.SetupTest(t)
	data := `{"name": "my_empty_baseline"}`
	w := CreateRequestRouterWithParams("PUT", "/", bytes.NewBufferString(data), nil, CreateBaselineHandler, 1, "PUT", "/")

	assert.Equal(t, http.StatusOK, w.Code)
	var resp CreateBaselineResponse
	ParseResponseBody(t, w.Body.Bytes(), &resp)
	database.CheckBaseline(t, resp.BaselineID, []string{}, "", "my_empty_baseline", "")
	database.DeleteBaseline(t, resp.BaselineID)
}

func TestCreateBaselineNameEmptyString(t *testing.T) {
	core.SetupTest(t)
	data := `{"name": ""}`
	w := CreateRequestRouterWithParams("PUT", "/", bytes.NewBufferString(data), nil, CreateBaselineHandler, 1, "PUT", "/")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, BaselineMissingNameErr, errResp.Error)
}

func TestCreateBaselineMissingName(t *testing.T) {
	core.SetupTest(t)
	data := `{}`
	w := CreateRequestRouterWithParams("PUT", "/", bytes.NewBufferString(data), nil, CreateBaselineHandler, 1, "PUT", "/")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, BaselineMissingNameErr, errResp.Error)
}

func TestCreateBaselineInvalidRequest(t *testing.T) {
	core.SetupTest(t)
	data := `{"name": 0}`
	w := CreateRequestRouterWithParams("PUT", "/", bytes.NewBufferString(data), nil, CreateBaselineHandler, 1, "PUT", "/")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.True(t, strings.Contains(errResp.Error,
		"cannot unmarshal number into Go struct field CreateBaselineRequest.name of type string"))
}

func TestCreateBaselineDuplicatedName(t *testing.T) {
	core.SetupTest(t)
	data := `{
		"name": "baseline_1-1"
	}`
	w := CreateRequestRouterWithParams("PUT", "/", bytes.NewBufferString(data), nil, CreateBaselineHandler, 1, "PUT", "/")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "baseline name already exists", errResp.Error)
}
