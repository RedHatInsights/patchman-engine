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

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var templateUUID = "11111111-2222-3333-4444-555555555555"
var templatePath = "/:template_id/systems"
var templateAccount = 1
var templateSystems = []uuid.UUID{
	uuid.MustParse("00000000-0000-0000-0000-000000000004"),
	uuid.MustParse("00000000-0000-0000-0000-000000000007"),
	uuid.MustParse("00000000-0000-0000-0000-000000000008"),
}

func TestUpdateTemplateSystems(t *testing.T) {
	core.SetupTest(t)
	data := `{
		"systems": [
			"00000000-0000-0000-0000-000000000004",
			"00000000-0000-0000-0000-000000000008"
		]
	}`

	database.CreateTemplate(t, templateAccount, templateUUID, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000007"),
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

	database.CreateTemplate(t, templateAccount, templateUUID, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000007"),
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
	database.CreateTemplate(t, templateAccount, templateUUID, []uuid.UUID{})
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
	database.CreateTemplate(t, templateAccount, templateUUID, []uuid.UUID{})
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
	database.CreateTemplate(t, templateAccount, template2, []uuid.UUID{})

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

	database.CheckTemplateSystems(t, templateAccount, template2, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000004"),
		uuid.MustParse("00000000-0000-0000-0000-000000000005"),
	})
	database.CheckTemplateSystems(t, templateAccount, templateUUID, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000007"),
		uuid.MustParse("00000000-0000-0000-0000-000000000008"),
	})
	database.DeleteTemplate(t, templateAccount, templateUUID)
	database.DeleteTemplate(t, templateAccount, template2)
}

func testUpdateTemplateBadRequest(t *testing.T, satelliteManaged, bootc bool) {
	core.SetupTestEnvironment()

	database.CreateTemplate(t, templateAccount, templateUUID, templateSystems)
	defer database.DeleteTemplate(t, templateAccount, templateUUID)

	inv := models.SystemInventory{
		InventoryID:      uuid.MustParse("99999999-0000-0000-0000-000000000015"),
		RhAccountID:      templateAccount,
		DisplayName:      "template_bad_request_test",
		Tags:             []byte("[]"),
		WorkspaceID:      database.TestWorkspace1IDPtr(),
		WorkspaceName:    database.TestWorkspace1NamePtr(),
		BuiltPkgcache:    true,
		SatelliteManaged: satelliteManaged,
		Bootc:            bootc,
	}
	assert.Nil(t, database.DB.Create(&inv).Error)
	assert.Nil(t, database.DB.Create(&models.SystemPatch{
		SystemID:    inv.ID,
		RhAccountID: templateAccount,
	}).Error)
	defer func() {
		assert.Nil(t, database.DB.Where("system_id = ? AND rh_account_id = ?", inv.ID, templateAccount).
			Delete(&models.SystemPatch{}).Error)
		assert.Nil(t, database.DB.Delete(&inv).Error)
	}()

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
	cases := []struct {
		name             string
		satelliteManaged bool
		bootc            bool
	}{
		{name: "satellite_system_test", satelliteManaged: true},
		{name: "bootc_system_test", bootc: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testUpdateTemplateBadRequest(t, tc.satelliteManaged, tc.bootc)
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

	database.CreateTemplate(t, templateAccount, templateUUID, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000007"),
	})
	defer database.DeleteTemplate(t, templateAccount, templateUUID)
	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount, core.ContextKV{Key: utils.KeyOrgID, Value: orgID})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	database.CheckTemplateSystems(t, templateAccount, templateUUID,
		[]uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000007")})

	// Expect HTTP 400 status code even when only 1 system causes candlepin call error
	data = `{
		"systems": [
			"00000000-0000-0000-0000-000000000004",
			"00000000-0000-0000-0000-000000000018"
		]
	}`
	w = CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBufferString(data), "",
		TemplateSystemsUpdateHandler, templateAccount, core.ContextKV{Key: utils.KeyOrgID, Value: orgID})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	database.CheckTemplateSystems(t, templateAccount, templateUUID,
		[]uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000007")})
}

func TestUpdateTemplateTooManySystems(t *testing.T) {
	core.SetupTest(t)

	database.CreateTemplate(t, templateAccount, templateUUID, templateSystems)
	defer database.DeleteTemplate(t, templateAccount, templateUUID)

	systems := make([]string, 0, TemplateSystemsUpdateLimit+1)
	for i := 0; i < TemplateSystemsUpdateLimit+1; i++ {
		systems = append(systems, uuid.NewString())
	}
	body := map[string][]string{
		"systems": systems,
	}

	bodyJSON, err := sonic.Marshal(body)
	if err != nil {
		panic(err)
	}

	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBuffer(bodyJSON), "",
		TemplateSystemsUpdateHandler, templateAccount)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, fmt.Sprintf("Cannot process more than %d systems at once", TemplateSystemsUpdateLimit), errResp.Error)
}
