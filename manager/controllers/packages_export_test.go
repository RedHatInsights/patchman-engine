package controllers

import (
	"app/base/core"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageExportJSON(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/", nil, "application/json", PackagesExportHandler, 3, "GET", "/")

	var output []PackageItem
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 4, len(output))
	assert.Equal(t, "kernel", output[0].Name)
}

func TestPackageExportCSV(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/", nil, "text/csv", PackagesExportHandler, 3, "GET", "/")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 6, len(lines))
	assert.Equal(t, "name,summary,systems_installed,systems_installable,systems_applicable", lines[0])

	assert.Equal(t, "kernel,The Linux kernel,3,2,2", lines[1])
}
