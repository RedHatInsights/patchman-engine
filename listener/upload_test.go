package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetOrCreateAccount(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	deleteData(t)

	accountId1 := getOrCreateTestAccount(t)
	accountId2 := getOrCreateTestAccount(t)
	assert.Equal(t, accountId1, accountId2)

	deleteData(t)
}

func TestUpdateSystemPlatform(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	deleteData(t)

	accountId := getOrCreateTestAccount(t)
	req := vmaas.UpdatesV3Request{
		PackageList:    []string{"package0"},
		RepositoryList: []string{},
		ModulesList:    []vmaas.UpdatesRequestModulesList{},
		Releasever:     "7Server",
		Basearch:       "x86_64",
	}
	sys, err := updateSystemPlatform(id, accountId, &req)
	assert.Nil(t, err)

	assertSystemInDb(t)

	sys2, err := updateSystemPlatform(id, accountId, &req)
	assert.Nil(t, err)

	assert.Equal(t, sys, sys2)

	deleteData(t)
}

func TestParseUploadMessage(t *testing.T) {
	event := createTestUploadEvent(t)
	identity, err := parseUploadMessage(&event)
	assert.Nil(t, err)
	assert.Equal(t, id, event.Id)
	assert.Equal(t, "User", identity.Identity.Type)

	ident, err := utils.Identity{
		Entitlements: nil,
		Identity:     utils.IdentityDetail{},
	}.Encode()
	assert.Nil(t, err)

	event.B64Identity = &ident
	_, err = parseUploadMessage(&event)
	assert.NotNil(t, err, "Should return not entitled error")

	ident = "Invalid"
	event.B64Identity = &ident
	_, err = parseUploadMessage(&event)
	assert.NotNil(t, err, "Should report invalid identity")

	event.B64Identity = nil
	_, err = parseUploadMessage(&event)
	assert.NotNil(t, err, "Should report missing identity")
}

func TestUploadHandler(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()
	deleteData(t)

	getOrCreateTestAccount(t)
	event := createTestUploadEvent(t)
	uploadHandler(event)

	assertSystemInDb(t)
	database.CheckSystemJustEvaluated(t, id, 3, 0, 0, 0)
	database.CheckAdvisoriesInDb(t, []string{"ER1", "ER2", "ER3"})
	database.CheckSystemAdvisoriesFirstReportedGreater(t, "2020-01-01", 3)
	deleteData(t)
}
