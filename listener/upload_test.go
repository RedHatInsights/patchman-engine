package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const id = "TEST-00000"

func deleteData(t *testing.T) {
	// Delete debug data from previous run
	assert.Nil(t, database.Db.Unscoped().Where("inventory_id = ?", id).Delete(&models.SystemPlatform{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("name = ?", id).Delete(&models.RhAccount{}).Error)
}

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
	msg := createTestingUploadKafkaMsg()
	event, identity, err := parseUploadMessage(msg)
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
	msg := createTestingUploadKafkaMsg()
	uploadHandler(msg)

	assertSystemInDb(t)

	deleteData(t)
}

func assertSystemInDb(t *testing.T) {
	var system models.SystemPlatform
	assert.Nil(t, database.Db.Where("inventory_id = ?", id).Find(&system).Error)
	assert.Equal(t, system.InventoryID, id)

	var account models.RhAccount
	assert.Nil(t, database.Db.Where("id = ?", system.RhAccountID).Find(&account).Error)
	assert.Equal(t, id, account.Name)

	now := time.Now().Add(-time.Minute)
	assert.True(t, system.FirstReported.After(now), "First reported")
	assert.True(t, system.LastUpdated.After(now), "Last updated")
	assert.True(t, system.UnchangedSince.After(now), "Unchanged since")
	assert.True(t, system.LastUpload.After(now), "Last upload")
	// Last eval should be nil, system has not yet been evaluated
	assert.Nil(t, system.LastEvaluation)
}

func getOrCreateTestAccount(t *testing.T) int {
	accountId, err := getOrCreateAccount(id)
	assert.Nil(t, err)
	return accountId
}

func createTestingUploadKafkaMsg() kafka.Message {
	msg := kafka.Message{Value: []byte(`{ "id": "TEST-00000", "b64_identity": "eyJlbnRpdGxlbWVudHMiOnsic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1ZX19LCJpZGVudGl0eSI6eyJhY2NvdW50X251bWJlciI6IlRFU1QtMDAwMDAiLCJ0eXBlIjoiVXNlciIsIkludGVybmFsIjpudWxsfX0="}`)}
	return msg
}
