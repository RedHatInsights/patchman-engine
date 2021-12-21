package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateBaseline(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	body := CreateBaselineRequst{
		Name:         "test_name",
		InventoryIDs: []SystemID{"00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000006"},
		ToTime:       "2021-01-01 12:00:00-04",
	}

	bodyJSON, err := json.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/", bytes.NewBuffer(bodyJSON))
	core.InitRouterWithParams(CreateBaselineHandler, 1, "PUT", "/").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output int
	ParseReponseBody(t, w.Body.Bytes(), &output)

	testUpdatedBaselineSystems(t, output)

	database.DeleteBaseline(t, 100, []string{
		"00000000-0000-0000-0000-000000000005",
		"00000000-0000-0000-0000-000000000006",
	})
}

func testUpdatedBaselineSystems(t *testing.T, temporaryBaselineID int) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	path := fmt.Sprintf(`/%v/systems`, temporaryBaselineID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", path, nil)
	core.InitRouterWithPath(BaselineSystemsListHandler, "/:baseline_id/systems").ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var output BaselineSystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)

	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "baseline_systems", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000006", output.Data[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000005", output.Data[1].ID)
}
