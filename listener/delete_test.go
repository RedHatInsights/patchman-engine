package listener

import (
	"app/base/core"
	"app/base/utils"
	"github.com/bmizerany/assert"
	log "github.com/sirupsen/logrus"
	"testing"
)

func TestDeleteSystem(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	deleteData(t)
	uploadEvent := createTestUploadEvent(id, true)
	uploadHandler(uploadEvent)
	assertSystemInDb(t, id, nil)

	deleteEvent := createTestDeleteEvent(id)
	deleteHandler(deleteEvent)
	assertSystemNotInDb(t)
}

func TestDeleteSystemWarn1(t *testing.T) {
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	deleteEvent := createTestDeleteEvent(id)
	deleteEvent.Type = nil
	deleteHandler(deleteEvent)
	assert.Equal(t, WarnEmptyEventType, logHook.LogEntries[len(logHook.LogEntries)-1].Message)
}

func TestDeleteSystemWarn2(t *testing.T) {
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	deleteEvent := createTestDeleteEvent(id)
	nonDeleteType := "no-delete"
	deleteEvent.Type = &nonDeleteType
	deleteHandler(deleteEvent)
	assert.Equal(t, WarnNoDeleteType, logHook.LogEntries[len(logHook.LogEntries)-1].Message)
}

func TestDeleteSystemWarn3(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)

	deleteEvent := createTestDeleteEvent("not-existing-id")
	deleteHandler(deleteEvent)

	assert.Equal(t, WarnNoRowsModified, logHook.LogEntries[len(logHook.LogEntries)-1].Message)
}
