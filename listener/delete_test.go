package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeleteSystem(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	deleteData(t)
	assert.NoError(t, database.Db.Create(&models.SystemPlatform{
		InventoryID: id,
		RhAccountID: 1,
		DisplayName: id,
	}).Error)

	deleteEvent := createTestDeleteEvent(id)
	err := deleteHandler(deleteEvent)
	assertSystemNotInDb(t)
	assert.NoError(t, err)
}

func TestDeleteSystemWarn1(t *testing.T) {
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	deleteEvent := createTestDeleteEvent(id)
	deleteEvent.Type = nil
	err := deleteHandler(deleteEvent)
	assert.Equal(t, WarnEmptyEventType, logHook.LogEntries[len(logHook.LogEntries)-1].Message)
	assert.NoError(t, err)
}

func TestDeleteSystemWarn2(t *testing.T) {
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	deleteEvent := createTestDeleteEvent(id)
	nonDeleteType := "no-delete"
	deleteEvent.Type = &nonDeleteType
	err := deleteHandler(deleteEvent)
	assert.Equal(t, WarnNoDeleteType, logHook.LogEntries[len(logHook.LogEntries)-1].Message)
	assert.NoError(t, err)
}

func TestDeleteSystemWarn3(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)

	deleteEvent := createTestDeleteEvent("not-existing-id")
	err := deleteHandler(deleteEvent)
	assert.NoError(t, err)

	assert.Equal(t, WarnNoRowsModified, logHook.LogEntries[len(logHook.LogEntries)-1].Message)
}

func TestUploadAfterDelete(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	uploadEvent := createTestUploadEvent(id, true)
	err := uploadHandler(uploadEvent)
	assert.NoError(t, err)
	assertSystemNotInDb(t)
}

func TestDeleteCleanup(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	assert.NoError(t, database.Db.Delete(&models.DeletedSystem{}).Error)
}
