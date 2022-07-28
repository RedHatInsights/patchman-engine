package controllers

import (
	"app/base/core"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemPackagesExportHandlerJSON(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/00000000-0000-0000-0000-000000000013/packages",
		nil, &contentTypeJSON, SystemPackagesExportHandler, 3, "GET", "/:inventory_id/packages")

	var output []SystemPackageInline
	ParseResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 4, len(output))
	assert.Equal(t, output[0].Name, "kernel")
	assert.Equal(t, output[0].EVRA, "5.6.13-200.fc31.x86_64")
	assert.Equal(t, output[0].LatestEVRA, "5.6.13-200.fc31.x86_64")
	assert.Equal(t, output[0].Summary, "The Linux kernel")
}

func TestSystemPackagesExportHandlerCSV(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/00000000-0000-0000-0000-000000000013/packages",
		nil, &contentTypeCSV, SystemPackagesExportHandler, 3, "GET", "/:inventory_id/packages")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 6, len(lines))
	assert.Equal(t, "name,evra,summary,description,updatable,latest_evra", lines[0])

	assert.Equal(t, "kernel,5.6.13-200.fc31.x86_64,The Linux kernel,The kernel meta package,false,"+
		"5.6.13-200.fc31.x86_64", lines[1])
	assert.Equal(t, "firefox,76.0.1-1.fc31.x86_64,Mozilla Firefox Web browser,Mozilla Firefox is an "+
		"open-source web browser...,true,76.0.1-1.fc31.x86_64", lines[2])
}

func TestSystemPackagesExportUnknown(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/unknownsystem/packages", nil, &contentTypeCSV,
		SystemPackagesExportHandler, 3, "GET", "/:inventory_id/packages")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
