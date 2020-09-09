package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUpdateSystem(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	deleteData(t)
	assert.NoError(t, database.Db.Create(&models.SystemPlatform{
		InventoryID: id,
		RhAccountID: 1,
		DisplayName: id,
	}).Error)

	ev := createTestUploadEvent(id, false)
	name := "TEST_NAME"
	ev.Host.DisplayName = &name
	ev.Host.SystemProfile.InstalledPackages = []string{"kernel"}
	assert.NoError(t, HandleUpload(ev))

	var system models.SystemPlatform
	assert.NoError(t, database.Db.Find(&system, "inventory_id = ?", id).Error)

	assert.Equal(t, name, system.DisplayName)
}

func TestDeleteSystem(t *testing.T) {
	deleteData(t)
	assert.NoError(t, database.Db.Create(&models.SystemPlatform{
		InventoryID: id,
		RhAccountID: 1,
		DisplayName: id,
	}).Error)

	deleteEvent := createTestDeleteEvent(id)
	err := HandleDelete(deleteEvent)
	assertSystemNotInDB(t)
	assert.NoError(t, err)
}

func TestDeleteSystemWarn1(t *testing.T) {
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	deleteEvent := createTestDeleteEvent(id)
	deleteEvent.Type = nil

	data, err := json.Marshal(deleteEvent)
	assert.NoError(t, err)

	err = EventsMessageHandler(mqueue.Message{Value: data})
	assert.Equal(t, WarnEmptyEventType, logHook.LogEntries[len(logHook.LogEntries)-1].Message)

	assert.NoError(t, err)
}

func TestDeleteSystemWarn2(t *testing.T) {
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	deleteEvent := createTestDeleteEvent(id)
	nonDeleteType := "no-delete"
	deleteEvent.Type = &nonDeleteType

	data, err := json.Marshal(deleteEvent)
	assert.NoError(t, err)

	err = EventsMessageHandler(mqueue.Message{Value: data})
	assert.Equal(t, WarnUnknownType, logHook.LogEntries[len(logHook.LogEntries)-1].Message)

	assert.NoError(t, err)
}

func TestDeleteSystemWarn3(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)

	deleteEvent := createTestDeleteEvent("not-existing-id")
	err := HandleDelete(deleteEvent)
	assert.NoError(t, err)

	assert.Equal(t, WarnNoRowsModified, logHook.LogEntries[len(logHook.LogEntries)-1].Message)
}

func TestUploadAfterDelete(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	uploadEvent := createTestUploadEvent(id, true)
	err := HandleUpload(uploadEvent)
	assert.NoError(t, err)
	assertSystemNotInDB(t)
}

func TestDeleteCleanup(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	assert.NoError(t, database.Db.Delete(&models.DeletedSystem{}).Error)
}
