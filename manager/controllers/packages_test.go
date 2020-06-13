package controllers

import (
	"app/base/core"
	"app/base/models"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/INV-12/packages", nil)
	core.InitRouterWithParams(SystemPackagesHandler, "3", "GET", "/:inventory_id/packages").
		ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output models.SystemPackageData
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Contains(t, output, "kernel")
	assert.Contains(t, output, "firefox")
	assert.Len(t, output["kernel"].Updates, 1)
	assert.Len(t, output["firefox"].Updates, 2)
}
