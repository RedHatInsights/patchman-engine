package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testAccountSystemCounts(t *testing.T, acc int, count int) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	var output SystemsResponse

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	core.InitRouterWithAccount(SystemsListHandler, "/", acc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	ParseReponseBody(t, w.Body.Bytes(), &output)
	// data
	assert.Equal(t, count, len(output.Data))
}

func TestMissingAccount(t *testing.T) {
	testAccountSystemCounts(t, 0, 0)
	testAccountSystemCounts(t, 1, 8)
	testAccountSystemCounts(t, 2, 3)
	testAccountSystemCounts(t, 3, 3)
	testAccountSystemCounts(t, 4, 0)
}
