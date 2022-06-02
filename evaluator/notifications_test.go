package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/mqueue"
	ntf "app/base/notification"
	"app/base/utils"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	inventoryID = "00000000-0000-0000-0000-000000000012"
)

func checkPayload(t *testing.T, notification ntf.Notification, idx int, name, advType, synopsis string) {
	payload, ok := notification.Events[idx].Payload.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, payload["advisory_name"], name)
	assert.Equal(t, payload["advisory_type"], advType)
	assert.Equal(t, payload["synopsis"], synopsis)
}

func TestAdvisoriesNotificationPublish(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()

	configure()
	loadCache()
	mockWriter := mqueue.MockKafkaWriter{}
	notificationsPublisher = &mockWriter

	expectedAddedAdvisories := []string{"RH-1", "RH-2"}
	expectedAdvisoryIDs := []int{1, 2}     // advisories expected to be paired to the system after evaluation
	oldSystemAdvisoryIDs := []int{1, 3, 4} // old advisories paired with the system

	database.DeleteSystemAdvisories(t, systemID, expectedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, expectedAdvisoryIDs)
	database.CreateSystemAdvisories(t, rhAccountID, systemID, oldSystemAdvisoryIDs, nil)
	database.CreateAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs, 1)
	database.CheckCachesValid(t)

	// do evaluate the system
	err := evaluateHandler(mqueue.PlatformEvent{
		SystemIDs:  []string{"00000000-0000-0000-0000-000000000012"},
		RequestIDs: []string{"request-2"},
		AccountID:  rhAccountID})
	assert.NoError(t, err)
	advisoryIDs := database.CheckAdvisoriesInDB(t, expectedAddedAdvisories)

	assert.Equal(t, 1, len(mockWriter.Messages))

	var notificationSent ntf.Notification
	assert.Nil(t, json.Unmarshal(mockWriter.Messages[0].Value, &notificationSent))
	checkPayload(t, notificationSent, 0, "RH-1", "enhancement", "adv-1-syn")
	checkPayload(t, notificationSent, 1, "RH-2", "bugfix", "adv-2-syn")

	database.DeleteSystemAdvisories(t, systemID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs)
}

func TestAdvisoriesNotificationMessage(t *testing.T) {
	events := make([]ntf.Event, 1)
	events[0] = ntf.Event{
		Payload: ntf.Advisory{
			AdvisoryName: "RH-1",
			AdvisoryType: "bugfix",
			Synopsis:     "Resolves some bug",
		},
	}

	notification := ntf.MakeNotification(rhAccountID, inventoryID, NewAdvisoryEvent, events)
	msg, err := mqueue.MessageFromJSON(inventoryID, notification)
	assert.Nil(t, err)
	assert.Equal(t, inventoryID, string(msg.Key))

	notificationJSON, err := json.Marshal(notification)
	assert.Nil(t, err)
	assert.Equal(t, notificationJSON, msg.Value)
}
