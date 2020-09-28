package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPackageSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/kernel/systems", nil)
	core.InitRouterWithParams(PackageSystemsListHandler, 3, "GET", "/:package_name/systems").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output PackageSystemsResponse
	assert.Greater(t, len(w.Body.Bytes()), 0)
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output.Data[0].InventoryID)
	assert.Equal(t, "5.6.13-200.fc31.x86_64", output.Data[1].EVRA)
}
