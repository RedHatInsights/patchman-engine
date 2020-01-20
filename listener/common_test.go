package listener

import (
	"app/base/database"
	"app/base/models"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const id = "TEST-00000"

func deleteData(t *testing.T) {
	// Delete test data from previous run
	assert.Nil(t, database.Db.Unscoped().Exec("DELETE FROM advisory_account_data aad "+
		"USING rh_account ra WHERE ra.id = aad.rh_account_id AND ra.name = ?", id).Error)
	assert.Nil(t, database.Db.Unscoped().Where("first_reported > timestamp '2020-01-01'").
		Delete(&models.SystemAdvisories{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("name IN ('ER1', 'ER2', 'ER3')").
		Delete(&models.AdvisoryMetadata{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("inventory_id = ?", id).Delete(&models.SystemPlatform{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("name = ?", id).Delete(&models.RhAccount{}).Error)
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
}

func assertSystemNotInDb(t *testing.T) {
	var systemCount int
	assert.Nil(t, database.Db.Model(models.SystemPlatform{}).
		Where("inventory_id = ?", id).Count(&systemCount).Error)

	assert.Equal(t, systemCount, 0)
}

func getOrCreateTestAccount(t *testing.T) int {
	accountID, err := getOrCreateAccount(id)
	assert.Nil(t, err)
	return accountID
}

func createTestUploadEvent(t *testing.T) PlatformEvent {
	msg := kafka.Message{Value: []byte(`{ "id": "TEST-00000","type": "created", "b64_identity": "eyJlbnRpdGxlbWVudHMiOnsic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1ZX19LCJpZGVudGl0eSI6eyJhY2NvdW50X251bWJlciI6IlRFU1QtMDAwMDAiLCJ0eXBlIjoiVXNlciIsIkludGVybmFsIjpudWxsfX0="}`)} //nolint:lll
	var event PlatformEvent
	err := json.Unmarshal(msg.Value, &event)
	assert.Nil(t, err)
	return event
}

func createTestDeleteEvent(t *testing.T) PlatformEvent {
	msg := kafka.Message{Value: []byte(`{ "id": "TEST-00000","type": "delete"}`)}
	var event PlatformEvent
	err := json.Unmarshal(msg.Value, &event)
	assert.Nil(t, err)
	return event
}

func TestParseEvents(t *testing.T) {
	msg := kafka.Message{Value: []byte(`{"id": "TEST-00000", "type": "delete"}`)}

	reached := false

	makeKafkaHandler(func(event PlatformEvent) {
		assert.Equal(t, event.ID, "TEST-00000")
		assert.Equal(t, *event.Type, "delete")
		reached = true
	})(msg)

	assert.True(t, reached, "Event handler should have been called")
}
