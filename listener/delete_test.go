package listener

import (
	"app/base/core"
	"app/base/utils"
	"testing"
)

func TestDeleteSystem(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	deleteData(t)
	uploadEvent := createTestUploadEvent(t, id)
	uploadHandler(uploadEvent)
	assertSystemInDb(t, id, nil)

	deleteEvent := createTestDeleteEvent(t)
	deleteHandler(deleteEvent)
	assertSystemNotInDb(t)
}
