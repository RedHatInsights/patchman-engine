package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaselineDelete(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	baselineID := database.CreateBaseline(t)
	path := fmt.Sprintf(`/%v`, baselineID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", path, nil)
	core.InitRouterWithPath(BaselineDeleteHandler, "/:baseline_id").ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	newpath := fmt.Sprintf(`/?filter[id]=%v`, baselineID)
	output := testBaselineAfterDelete(t, newpath)

	assert.Equal(t, 0, len(output.Data))
}

func testBaselineAfterDelete(t *testing.T, url string) BaselinesResponse {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	core.InitRouter(BaselinesListHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var output BaselinesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)

	return output
}
