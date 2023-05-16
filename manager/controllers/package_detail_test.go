package controllers

import (
	"app/base/core"
	"app/base/utils"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// nolint: dupl
func TestLatestPackage(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/packages/kernel", nil, "", PackageDetailHandler, 3,
		"GET", "/packages/:package_name")

	var output PackageDetailResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, "kernel", output.Data.Attributes.Name)
	assert.Equal(t, "The Linux kernel", output.Data.Attributes.Summary)
	assert.Equal(t, "The kernel meta package", output.Data.Attributes.Description)
	assert.Equal(t, "0:5.6.13-201.fc31.x86_64", output.Data.Attributes.EVRA)
	assert.Equal(t, "RH-7", output.Data.Attributes.AdvID)
	assert.Equal(t, "kernel-0:5.6.13-201.fc31.x86_64", output.Data.ID)
	assert.Equal(t, "package", output.Data.Type)
}

// nolint: dupl
func TestEvraPackage(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/packages/kernel-0:5.6.13-200.fc31.x86_64", nil, "",
		PackageDetailHandler, 3, "GET", "/packages/:package_name")

	var output PackageDetailResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, "kernel", output.Data.Attributes.Name)
	assert.Equal(t, "The Linux kernel", output.Data.Attributes.Summary)
	assert.Equal(t, "The kernel meta package", output.Data.Attributes.Description)
	assert.Equal(t, "0:5.6.13-200.fc31.x86_64", output.Data.Attributes.EVRA)
	assert.Equal(t, "RH-1", output.Data.Attributes.AdvID)
	assert.Equal(t, "kernel-0:5.6.13-200.fc31.x86_64", output.Data.ID)
	assert.Equal(t, "package", output.Data.Type)
}

func TestNonExitentPackage(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/packages/python", nil, "", PackageDetailHandler, 3,
		"GET", "/packages/:package_name")

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, "invalid package name", errResp.Error)
}

func TestNonExitentEvra(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/packages/kernel-5.6.13-202.fc31.x86_64", nil, "",
		PackageDetailHandler, 3, "GET", "/packages/:package_name")

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusNotFound, &errResp)
	assert.Equal(t, "package not found", errResp.Error)
}

func TestPackageDetailNoPackage(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequest("GET", "/", nil, "", PackageDetailHandler)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, "package_param not found", errResp.Error)
}

func TestPackageDetailFiltering(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/packages/kernel-5.6.13-202.fc31.x86_64?filter[filter]=abcd",
		nil, "", PackageDetailHandler, 3, "GET", "/packages/:package_name")

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, FilterNotSupportedMsg, errResp.Error)
}
