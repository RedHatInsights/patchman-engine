package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testBaselineDetail(t *testing.T, url string, expectedStatus int, output interface{}) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", url, nil, "", BaselineDetailHandler, "/:baseline_id")
	CheckResponse(t, w, expectedStatus, &output)
}

func TestBaselineDetailDefault(t *testing.T) {
	var output BaselineDetailResponse
	testBaselineDetail(t, "/1", http.StatusOK, &output)
	assert.Equal(t, int64(1), output.Data.ID)
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
	core.SetupTest(t)
	baselineID := database.CreateBaselineWithConfig(t, "", nil, nil, nil)
	var output BaselineDetailResponse
	url := fmt.Sprintf("/%d", baselineID)
	testBaselineDetail(t, url, http.StatusOK, &output)
	assert.Equal(t, baselineID, output.Data.ID)
	assert.Equal(t, "baseline", output.Data.Type)
	assert.Equal(t, "temporary_baseline", output.Data.Attributes.Name)
	assert.Nil(t, output.Data.Attributes.Config)
	database.DeleteBaseline(t, baselineID)
}
