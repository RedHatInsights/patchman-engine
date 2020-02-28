package listener

import (
	"app/base/core"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetOrCreateAccount(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	deleteData(t)

	accountID1 := getOrCreateTestAccount(t)
	accountID2 := getOrCreateTestAccount(t)
	assert.Equal(t, accountID1, accountID2)

	deleteData(t)
}

func createTestInvHost() *Host {
	correctTimestamp := "2018-09-22T12:00:00-04:00"
	wrongTimestamp := "x018-09-22T12:00:00-04:00"

	host := Host{StaleTimestamp: &correctTimestamp, StaleWarningTimestamp: &wrongTimestamp}
	return &host
}

func TestUpdateSystemPlatform(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	deleteData(t)

	accountID1 := getOrCreateTestAccount(t)
	accountID2 := getOrCreateTestAccount(t)
	req := vmaas.UpdatesV3Request{
		PackageList:    []string{"package0"},
		RepositoryList: []string{},
		ModulesList:    []vmaas.UpdatesRequestModulesList{},
		Releasever:     "7Server",
		Basearch:       "x86_64",
	}

	sys1, err := updateSystemPlatform(id, accountID1, createTestInvHost(), &req)
	assert.Nil(t, err)

	assertSystemInDb(t, id, &accountID1)

	sys2, err := updateSystemPlatform(id, accountID2, createTestInvHost(), &req)
	assert.Nil(t, err)

	assertSystemInDb(t, id, &accountID2)

	assert.Equal(t, sys1.ID, sys2.ID)
	assert.Equal(t, sys1.InventoryID, sys2.InventoryID)
	assert.Equal(t, sys1.JSONChecksum, sys2.JSONChecksum)
	assert.Equal(t, sys1.OptOut, sys2.OptOut)
	assert.NotNil(t, sys1.StaleTimestamp)
	assert.Nil(t, sys1.StaleWarningTimestamp)
	assert.Nil(t, sys1.CulledTimestamp)

	deleteData(t)
}

func TestUploadHandler(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()
	deleteData(t)

	_ = getOrCreateTestAccount(t)
	event := createTestUploadEvent(id, true)
	uploadHandler(event)

	assertSystemInDb(t, id, nil)
	deleteData(t)
}

func TestEmptyUploadHandler(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()

	logHook := utils.TestLogHook{}
	log.AddHook(&logHook)

	_ = getOrCreateTestAccount(t)
	inventoryID := "TEST-NO-PKGS"
	event := createTestUploadEvent(inventoryID, false)
	uploadHandler(event)

	assert.Equal(t, logHook.LogEntries[len(logHook.LogEntries)-1].Message, "skipping profile with no packages")
}
