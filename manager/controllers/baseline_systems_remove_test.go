package controllers

import (
	"app/base/core"
	"app/base/database"
	"bytes"
	"net/http"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
)

func testBaselineSystemsRemove(t *testing.T, body BaselineSystemsRemoveRequest, status int) {
	bodyJSON, err := sonic.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := CreateRequestRouterWithParams("POST", "/systems/remove", "", "", bytes.NewBuffer(bodyJSON), "",
		BaselineSystemsRemoveHandler, 1)

	assert.Equal(t, status, w.Code)
}

func TestBaselineSystemsRemoveDefault(t *testing.T) {
	core.SetupTest(t)

	var inventoryIDs = []string{
		"00000000-0000-0000-0000-000000000004",
		"00000000-0000-0000-0000-000000000005",
	}
	baselineID := database.CreateBaseline(t, "temporary_baseline_1", inventoryIDs, nil)

	var inventoryIDs2 = []string{
		"00000000-0000-0000-0000-000000000006",
	}
	baselineID2 := database.CreateBaseline(t, "temporary_baseline_2", inventoryIDs2, nil)

	req := BaselineSystemsRemoveRequest{
		InventoryIDs: append(inventoryIDs, inventoryIDs2...),
	}

	testBaselineSystemsRemove(t, req, http.StatusOK)
	database.DeleteBaseline(t, baselineID)
	database.DeleteBaseline(t, baselineID2)
	database.CheckBaselineDeleted(t, baselineID)
	database.CheckBaselineDeleted(t, baselineID2)
}

func TestBaselineSystemsRemoveInvalid(t *testing.T) {
	core.SetupTest(t)

	req1 := BaselineSystemsRemoveRequest{InventoryIDs: []string{}}
	req2 := BaselineSystemsRemoveRequest{InventoryIDs: []string{"foo"}}
	req3 := BaselineSystemsRemoveRequest{}

	testBaselineSystemsRemove(t, req1, http.StatusBadRequest)
	testBaselineSystemsRemove(t, req2, http.StatusBadRequest)
	testBaselineSystemsRemove(t, req3, http.StatusBadRequest)
}
