package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var templateUUID = "11111111-2222-3333-4444-555555555555"
var templatePath = "/:template_id/systems"
var templateAccount = 1
var templateSystems = []string{
	"00000000-0000-0000-0000-000000000004",
	"00000000-0000-0000-0000-000000000007",
	"00000000-0000-0000-0000-000000000008",
}

func TestUpdateTemplateSystems(t *testing.T) {
	core.SetupTest(t)
	data := `{
		"systems": [
			"00000000-0000-0000-0000-000000000004",
			"00000000-0000-0000-0000-000000000008"
		]
	}`

	database.CreateTemplate(t, templateAccount, templateUUID, []string{
		"00000000-0000-0000-0000-000000000007",
	})
	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)

	assert.Equal(t, http.StatusOK, w.Code)
	database.CheckTemplateSystems(t, templateAccount, templateUUID, templateSystems)
	database.DeleteTemplate(t, templateAccount, templateUUID)
}

func TestUpdateTemplateInvalidVersion(t *testing.T) {
	core.SetupTest(t)
	data := `{
		"systems": [
			"00000000-0000-0000-0000-000000000006"
		]
	}`

	database.CreateTemplate(t, templateAccount, templateUUID, []string{
		"00000000-0000-0000-0000-000000000007",
	})
	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	// 00000000-0000-0000-0000-000000000006 is RHEL 7
	assert.Equal(t, "Incompatible template and system version or architecture: template arch: x86_64, version: 8\n"+
		"system uuid: 00000000-0000-0000-0000-000000000006, arch: x86_64, version: 7",
		errResp.Error)
	database.DeleteTemplate(t, templateAccount, templateUUID)
}

func TestUpdateTemplateInvalidSystem(t *testing.T) {
	core.SetupTest(t)

	data := `{
		"systems": [
			"00000000-0000-0000-0000-000000000005",
			"c0ffeec0-ffee-c0ff-eec0-ffeec0ffee00",
			"00000000-0000-0000-0000-000000000009"
		]
	}`
	database.CreateTemplate(t, templateAccount, templateUUID, []string{})
	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusNotFound, &errResp)
	// 00000000-0000-0000-0000-000000000009 is from different account and should not be added
	// c0ffeec0-ffee-c0ff-eec0-ffeec0ffee00 is not a valid uuid
	assert.Equal(t, "not found\n"+
		"unknown inventory_ids: [00000000-0000-0000-0000-000000000009 c0ffeec0-ffee-c0ff-eec0-ffeec0ffee00]",
		errResp.Error)
	database.DeleteTemplate(t, templateAccount, templateUUID)
}

func TestUpdateTemplateSystemNotInCandlepin(t *testing.T) {
	core.SetupTest(t)

	data := `{
		"systems": [
			"00000000-0000-0000-0000-000000000005",
			"00000000-0000-0000-0000-000000000003"
		]
	}`
	database.CreateTemplate(t, templateAccount, templateUUID, []string{})
	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	// 00000000-0000-0000-0000-000000000003 is not in candlepin
	assert.Equal(t, "missing owner_id for systems\n'00000000-0000-0000-0000-000000000003'",
		errResp.Error)
	database.DeleteTemplate(t, templateAccount, templateUUID)
}

func TestUpdateTemplateNullValues(t *testing.T) {
	core.SetupTest(t)

	database.CreateTemplate(t, templateAccount, templateUUID, templateSystems)
	data := "{}"
	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)

	var resp interface{}
	CheckResponse(t, w, http.StatusBadRequest, &resp)
	database.CheckTemplateSystems(t, templateAccount, templateUUID, templateSystems)

	database.DeleteTemplate(t, templateAccount, templateUUID)
}

func TestUpdateTemplateInvalidTemplateID(t *testing.T) {
	core.SetupTestEnvironment()
	w := CreateRequestRouterWithParams("PUT", templatePath, "invalidTemplate", "", bytes.NewBufferString("{}"), "",
		TemplateSystemsUpdateHandler, templateAccount)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusNotFound, &errResp)
	assert.Equal(t, "Invalid template uuid: invalidTemplate", errResp.Error)
}

func TestReassignTemplateSystems2(t *testing.T) {
	core.SetupTestEnvironment()

	template2 := "99999999-9999-8888-8888-888888888888"
	database.CreateTemplate(t, templateAccount, templateUUID, templateSystems)
	database.CreateTemplate(t, templateAccount, template2, []string{})

	// Reassigning inventory IDs of another templates is allowed
	data := `{
		"systems": [
			"00000000-0000-0000-0000-000000000004",
			"00000000-0000-0000-0000-000000000005"
		]
	}`
	w := CreateRequestRouterWithParams("PUT", templatePath, template2, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)

	assert.Equal(t, http.StatusOK, w.Code)

	database.CheckTemplateSystems(t, templateAccount, template2, []string{
		"00000000-0000-0000-0000-000000000004",
		"00000000-0000-0000-0000-000000000005",
	})
	database.CheckTemplateSystems(t, templateAccount, templateUUID, []string{
		"00000000-0000-0000-0000-000000000007",
		"00000000-0000-0000-0000-000000000008",
	})
	database.DeleteTemplate(t, templateAccount, templateUUID)
	database.DeleteTemplate(t, templateAccount, template2)
}

func testUpdateTemplateBadRequest(t *testing.T, system models.SystemPlatform) {
	core.SetupTestEnvironment()

	database.CreateTemplate(t, templateAccount, templateUUID, templateSystems)
	defer database.DeleteTemplate(t, templateAccount, templateUUID)

	tx := database.DB.Create(&system)
	assert.Nil(t, tx.Error)
	defer database.DB.Delete(system)

	data := `{
		"systems": {
			"00000000-0000-0000-0000-000000000009",
			"99999999-0000-0000-0000-000000000015"
		}
	}`

	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)
	var err utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &err)
}

func TestUpdateTemplateBadRequest(t *testing.T) {
	system := models.SystemPlatform{
		InventoryID:   "99999999-0000-0000-0000-000000000015",
		RhAccountID:   templateAccount,
		BuiltPkgcache: true,
	}
	satelliteSystem := system
	satelliteSystem.DisplayName = "satellite_system_test"
	satelliteSystem.SatelliteManaged = true

	bootcSystem := system
	bootcSystem.DisplayName = "bootc_system_test"
	bootcSystem.Bootc = true

	systems := []models.SystemPlatform{satelliteSystem, bootcSystem}
	for _, system := range systems {
		t.Run(fmt.Sprint(system.DisplayName), func(t *testing.T) {
			testUpdateTemplateBadRequest(t, system)
		})
	}
}

func TestUpdateTemplateSystemsCandlepin404(t *testing.T) {
	core.SetupTest(t)
	// 00000000-0000-0000-0000-000000000018 will force candlepin mock to return 404
	// because owner_id=return_404
	data := `{
		"systems": [
			"00000000-0000-0000-0000-000000000018"
		]
	}`

	database.CreateTemplate(t, templateAccount, templateUUID, []string{
		"00000000-0000-0000-0000-000000000007",
	})
	defer database.DeleteTemplate(t, templateAccount, templateUUID)
	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)

	assert.Equal(t, http.StatusFailedDependency, w.Code)
	database.CheckTemplateSystems(t, templateAccount, templateUUID, []string{"00000000-0000-0000-0000-000000000007"})

	// Expect HTTP 200 status code when only 1 system causes candlepin call error
	data = `{
		"systems": [
			"00000000-0000-0000-0000-000000000004",
			"00000000-0000-0000-0000-000000000018"
		]
	}`
	w = CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)

	assert.Equal(t, http.StatusOK, w.Code)
	database.CheckTemplateSystems(t, templateAccount, templateUUID,
		[]string{"00000000-0000-0000-0000-000000000004", "00000000-0000-0000-0000-000000000007"})
}
