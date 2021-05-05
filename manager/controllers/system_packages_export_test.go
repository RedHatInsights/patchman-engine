package controllers

import (
	"app/base/core"
	"app/base/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemPackagesExportHandlerJSON(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000013/packages", nil)
	req.Header.Add("Accept", "application/json")
	core.InitRouterWithParams(SystemPackagesExportHandler, 3, "GET", "/:inventory_id/packages").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output []SystemPackageInline
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 4, len(output))
	assert.Equal(t, output[0].Name, "kernel")
	assert.Equal(t, output[0].EVRA, "5.6.13-200.fc31.x86_64")
	assert.Equal(t, output[0].LatestEVRA, "5.6.13-200.fc31.x86_64")
	assert.Equal(t, output[0].Summary, "The Linux kernel")
}

func TestSystemPackagesExportHandlerCSV(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000013/packages", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouterWithParams(SystemPackagesExportHandler, 3, "GET", "/:inventory_id/packages").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 6, len(lines))
	assert.Equal(t, "name,evra,summary,description,updatable,latest_evra", lines[0])

	assert.Equal(t, "kernel,5.6.13-200.fc31.x86_64,The Linux kernel,The kernel meta package,false,"+
		"5.6.13-200.fc31.x86_64", lines[1])
	assert.Equal(t, "firefox,76.0.1-1.fc31.x86_64,Mozilla Firefox Web browser,Mozilla Firefox is an "+
		"open-source web browser...,true,76.0.1-1.fc31.x86_64", lines[2])
}
