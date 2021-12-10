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

	assert.Equal(t, http.StatusOK, w.Code)
	var output []PackageSystemItem
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output[0].ID)
	assert.Equal(t, "5.6.13-200.fc31.x86_64", output[0].InstalledEVRA)
	assert.Equal(t, "5.10.13-200.fc31.x86_64", output[0].AvailableEVRA)
	assert.Equal(t, SystemTagsList{{"k1", "ns1", "val1"}}, output[0].Tags)
	assert.Equal(t, "", output[0].BaselineName)
	assert.Equal(t, utils.PtrBoolNil(), output[0].BaselineUpToDate)
}

func TestPackageSystemsExportHandlerCSV(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/kernel/systems?sort=id", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouterWithParams(PackageSystemsExportHandler, 3, "GET", "/:package_name/systems").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 4, len(lines))
	assert.Equal(t, "id,display_name,installed_evra,available_evra,updatable,tags",
		lines[0]) // TODO: ,baseline_name,baseline_uptodate
	assert.Equal(t, "00000000-0000-0000-0000-000000000012,00000000-0000-0000-0000-000000000012,"+
		"5.6.13-200.fc31.x86_64,5.10.13-200.fc31.x86_64,true,\"[{'key':'k1','namespace':'ns1','value':'val1'}]\"",
		lines[1]) // TODO: ,,
	assert.Equal(t, "00000000-0000-0000-0000-000000000013,00000000-0000-0000-0000-000000000013,"+
		"5.6.13-200.fc31.x86_64,,false,\"[{'key':'k1','namespace':'ns1','value':'val1'}]\"", lines[2]) // TODO: ,,
}

func TestPackageSystemsExportInvalidName(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unknown_package/systems", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouterWithParams(PackageSystemsExportHandler, 3, "GET", "/:package_name/systems").
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
