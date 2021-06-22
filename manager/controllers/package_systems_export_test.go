package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPackageSystemsExportHandlerJSON(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/kernel/systems?sort=id", nil)
	req.Header.Add("Accept", "application/json")
	core.InitRouterWithParams(PackageSystemsExportHandler, 3, "GET", "/:package_name/systems").
		ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output []PackageSystemItem
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output))
	assert.Equal(t, output[0].ID, "00000000-0000-0000-0000-000000000012")
	assert.Equal(t, output[0].InstalledEVRA, "5.6.13-200.fc31.x86_64")
	assert.Equal(t, output[0].AvailableEVRA, "5.10.13-200.fc31.x86_64")
}

func TestPackageSystemsExportHandlerCSV(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/kernel/systems?sort=id", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouterWithParams(PackageSystemsExportHandler, 3, "GET", "/:package_name/systems").
		ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 4, len(lines))
	assert.Equal(t, "id,display_name,installed_evra,available_evra,updatable", lines[0])
	assert.Equal(t, "00000000-0000-0000-0000-000000000012,00000000-0000-0000-0000-000000000012,"+
		"5.6.13-200.fc31.x86_64,5.10.13-200.fc31.x86_64,true",
		lines[1])
	assert.Equal(t, "00000000-0000-0000-0000-000000000013,00000000-0000-0000-0000-000000000013,"+
		"5.6.13-200.fc31.x86_64,,false", lines[2])
}
