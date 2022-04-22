package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testBaselineSystemsRemove(t *testing.T, body BaselineSystemsRemoveRequest) BaselineSystemsRemoveResponse {
	bodyJSON, err := json.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/systems/remove", bytes.NewBuffer(bodyJSON))
	core.InitRouterWithParams(BaselineSystemsRemoveHandler, 1, "POST", "/systems/remove").ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var output BaselineSystemsRemoveResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)

	return output
}

func TestBaselineSystemsRemoveDefault(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

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
	resp := testBaselineSystemsRemove(t, req)

	assert.Equal(t, len(req.InventoryIDs), len(resp.InventoryIDs))
	for i, ID := range resp.InventoryIDs {
		assert.Equal(t, req.InventoryIDs[i], ID)
	}

	database.DeleteBaseline(t, baselineID)
	database.DeleteBaseline(t, baselineID2)
	database.CheckBaselineDeleted(t, baselineID)
	database.CheckBaselineDeleted(t, baselineID2)
}
