package controllers

import (
	"app/base/database"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testBaselineSystemsRemove(t *testing.T, body BaselineSystemsRemoveRequest) {
	bodyJSON, err := json.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := CreateRequestRouterWithParams("POST", "/systems/remove", bytes.NewBuffer(bodyJSON), nil,
		BaselineSystemsRemoveHandler, 1, "POST", "/systems/remove")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBaselineSystemsRemoveDefault(t *testing.T) {
	SetupTest(t)

	var inventoryIDs = []string{
		"00000000-0000-0000-0000-000000000004",
		"00000000-0000-0000-0000-000000000005",
	}
	baselineID := database.CreateBaseline(t, "temporary_baseline_1", inventoryIDs)

	var inventoryIDs2 = []string{
		"00000000-0000-0000-0000-000000000006",
	}
	baselineID2 := database.CreateBaseline(t, "temporary_baseline_2", inventoryIDs2)

	req := BaselineSystemsRemoveRequest{
		InventoryIDs: append(inventoryIDs, inventoryIDs2...),
	}

	testBaselineSystemsRemove(t, req)
	database.DeleteBaseline(t, baselineID)
	database.DeleteBaseline(t, baselineID2)
	database.CheckBaselineDeleted(t, baselineID)
	database.CheckBaselineDeleted(t, baselineID2)
}
