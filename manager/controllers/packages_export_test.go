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

func TestPackageExportJSON(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "application/json")
	core.InitRouterWithParams(PackagesExportHandler, 3, "GET", "/").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output []PackageItem

	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 4, len(output))
	assert.Equal(t, "kernel", output[0].Name)
}

func TestPackageExportCSV(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouterWithParams(PackagesExportHandler, 3, "GET", "/").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 6, len(lines))
	assert.Equal(t, "name,systems_installed,systems_updatable,summary", lines[0])

	assert.Equal(t, "kernel,2,1,The Linux kernel", lines[1])
}
