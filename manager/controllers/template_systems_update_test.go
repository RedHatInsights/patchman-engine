package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var templateUUID = "11111111-2222-3333-4444-555555555555"
var templatePath = "/:template_id/systems"
var templateAccount = 1
var templateSystems = []string{
	"00000000-0000-0000-0000-000000000004",
	"00000000-0000-0000-0000-000000000006",
	"00000000-0000-0000-0000-000000000007",
	"00000000-0000-0000-0000-000000000008",
}

func TestUpdateTemplateSystems(t *testing.T) {
	core.SetupTest(t)
	data := `{
		"systems": [
			"00000000-0000-0000-0000-000000000004",
			"00000000-0000-0000-0000-000000000006",
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

func TestUpdateTemplateInvalidSystem(t *testing.T) {
	core.SetupTest(t)

	data := `{
		"systems": [
			"00000000-0000-0000-0000-000000000005",
			"this-is-not-a-uuid",
			"00000000-0000-0000-0000-000000000009"
		]
	}`
	database.CreateTemplate(t, templateAccount, templateUUID, []string{})
	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusNotFound, &errResp)
	// 00000000-0000-0000-0000-000000000009 is from different account and should not be added
	// this-is-not-a-uuid is not a valid uuid
	assert.Equal(t, "Unknown inventory_ids: [00000000-0000-0000-0000-000000000009 this-is-not-a-uuid]",
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
			"00000000-0000-0000-0000-000000000005",
			"00000000-0000-0000-0000-000000000006"
		]
	}`
	w := CreateRequestRouterWithParams("PUT", templatePath, template2, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount)

	assert.Equal(t, http.StatusOK, w.Code)

	database.CheckTemplateSystems(t, templateAccount, template2, []string{
		"00000000-0000-0000-0000-000000000004",
		"00000000-0000-0000-0000-000000000005",
		"00000000-0000-0000-0000-000000000006",
	})
	database.CheckTemplateSystems(t, templateAccount, templateUUID, []string{
		"00000000-0000-0000-0000-000000000007",
		"00000000-0000-0000-0000-000000000008",
	})
	database.DeleteTemplate(t, templateAccount, templateUUID)
	database.DeleteTemplate(t, templateAccount, template2)
}

func TestUpdateTemplateSatelliteSystem(t *testing.T) {
	core.SetupTestEnvironment()

	database.CreateTemplate(t, templateAccount, templateUUID, templateSystems)
	defer database.DeleteTemplate(t, templateAccount, templateUUID)

	system := models.SystemPlatform{
		InventoryID:      "99999999-0000-0000-0000-000000000015",
		DisplayName:      "satellite_system_test",
		RhAccountID:      templateAccount,
		BuiltPkgcache:    true,
		SatelliteManaged: true,
	}
	tx := database.Db.Create(&system)
	assert.Nil(t, tx.Error)
	defer database.Db.Delete(system)

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
