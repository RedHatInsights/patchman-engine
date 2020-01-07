package listener

import (
	"app/base/core"
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
	req := vmaas.UpdatesRequest{
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
}

func TestUploadHandler(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()
	deleteData(t)

	getOrCreateTestAccount(t)
	event := createTestUploadEvent(t)
	uploadHandler(event)

	assertSystemInDb(t)

	deleteData(t)
}

