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
	core.InitRouterWithParams(PackageSystemsListHandler, "3", "GET", "/:package_name/systems").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output []string
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 1, len(output))
	assert.Equal(t, "INV-12", output[0])
}
