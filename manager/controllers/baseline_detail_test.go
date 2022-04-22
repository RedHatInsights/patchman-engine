package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testBaselineDetail(t *testing.T, url string, expectedStatus int, output interface{}) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	core.InitRouterWithPath(BaselineDetailHandler, "/:baseline_id").ServeHTTP(w, req)

	assert.Equal(t, expectedStatus, w.Code)
	ParseResponseBody(t, w.Body.Bytes(), &output)
}

func TestBaselineDetailDefault(t *testing.T) {
	var output BaselineDetailResponse
	testBaselineDetail(t, "/1", http.StatusOK, &output)
	assert.Equal(t, 1, output.Data.ID)
	assert.Equal(t, "baseline", output.Data.Type)
	assert.Equal(t, "baseline_1-1", output.Data.Attributes.Name)
	assert.Equal(t, "2010-09-22T00:00:00Z",
		output.Data.Attributes.Config.ToTime.Format(time.RFC3339))
	assert.Equal(t, "desc", output.Data.Attributes.Description)
}

func TestBaselineDetailNotFound(t *testing.T) {
	var output utils.ErrorResponse
	testBaselineDetail(t, "/10000", http.StatusNotFound, &output)
	assert.Equal(t, "baseline not found", output.Error)
}

func TestBaselineDetailInvalid(t *testing.T) {
	var output utils.ErrorResponse
	testBaselineDetail(t, "/invalidID", http.StatusBadRequest, &output)
	assert.Equal(t, "Invalid baseline_id: invalidID", output.Error)
}

func TestBaselineDetailEmptyConfig(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	baselineID := database.CreateBaselineWithConfig(t, "", nil, nil)
	var output BaselineDetailResponse
	url := fmt.Sprintf("/%d", baselineID)
	testBaselineDetail(t, url, http.StatusOK, &output)
	assert.Equal(t, baselineID, output.Data.ID)
	assert.Equal(t, "baseline", output.Data.Type)
	assert.Equal(t, "temporary_baseline", output.Data.Attributes.Name)
	assert.Nil(t, output.Data.Attributes.Config)
	database.DeleteBaseline(t, baselineID)
}
