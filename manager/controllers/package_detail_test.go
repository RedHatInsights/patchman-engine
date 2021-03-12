package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

//nolint: dupl
func TestLatestPackage(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/packages/kernel", nil)
	core.InitRouterWithParams(PackageDetailHandler, 3, "GET", "/packages/:package_name").
		ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var output PackageDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, "kernel", output.Data.Attributes.Name)
	assert.Equal(t, "The Linux kernel", output.Data.Attributes.Summary)
	assert.Equal(t, "The kernel meta package", output.Data.Attributes.Description)
	assert.Equal(t, "5.6.13-201.fc31.x86_64", output.Data.Attributes.EVRA)
	assert.Equal(t, "RH-7", output.Data.Attributes.AdvID)
	assert.Equal(t, "kernel-5.6.13-201.fc31.x86_64", output.Data.ID)
	assert.Equal(t, "package", output.Data.Type)
}

//nolint: dupl
func TestEvraPackage(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/packages/kernel-5.6.13-200.fc31.x86_64", nil)
	core.InitRouterWithParams(PackageDetailHandler, 3, "GET", "/packages/:package_name").
		ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var output PackageDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, "kernel", output.Data.Attributes.Name)
	assert.Equal(t, "The Linux kernel", output.Data.Attributes.Summary)
	assert.Equal(t, "The kernel meta package", output.Data.Attributes.Description)
	assert.Equal(t, "5.6.13-200.fc31.x86_64", output.Data.Attributes.EVRA)
	assert.Equal(t, "RH-1", output.Data.Attributes.AdvID)
	assert.Equal(t, "kernel-5.6.13-200.fc31.x86_64", output.Data.ID)
	assert.Equal(t, "package", output.Data.Type)
}

func TestNonExitentPackage(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/packages/python", nil)
	core.InitRouterWithParams(PackageDetailHandler, 3, "GET", "/packages/:package_name").
		ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid package name", errResp.Error)
}

func TestNonExitentEvra(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/packages/kernel-5.6.13-202.fc31.x86_64", nil)
	core.InitRouterWithParams(PackageDetailHandler, 3, "GET", "/packages/:package_name").
		ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "package not found", errResp.Error)
}

func TestPackageDetailNoPackage(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	core.InitRouter(PackageDetailHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "package_param not found", errResp.Error)
}
