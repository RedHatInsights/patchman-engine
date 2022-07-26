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
	w := CreateRequestRouterWithParams("GET", "/", nil, &contentTypeJSON, PackagesExportHandler, 3, "GET", "/")

	var output []PackageItem
	ParseResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 4, len(output))
	assert.Equal(t, "kernel", output[0].Name)
}

func TestPackageExportCSV(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/", nil, &contentTypeCSV, PackagesExportHandler, 3, "GET", "/")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 6, len(lines))
	assert.Equal(t, "name,systems_installed,systems_updatable,summary", lines[0])

	assert.Equal(t, "kernel,2,1,The Linux kernel", lines[1])
}
