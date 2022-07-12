package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/mqueue"
	ntf "app/base/notification"
	"app/base/utils"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	inventoryID = "00000000-0000-0000-0000-000000000012"
)

func checkNotificationPayload(t *testing.T, notification ntf.Notification, name, advType, synopsis string) {
	for _, event := range notification.Events {
		payload, ok := event.Payload.(map[string]interface{})
		assert.True(t, ok)

		if payload["advisory_name"] != name {
			continue
		}
		if payload["advisory_type"] != advType {
			continue
		}
		if payload["synopsis"] != synopsis {
			continue
		}
		return
	}
	t.Fatal("such payload does not exist")
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
	database.CheckAdvisoriesAccountDataNotified(t, rhAccountID, oldSystemAdvisoryIDs, false)

	orgID := "1234567"
	// do evaluate the system
	err := evaluateHandler(mqueue.PlatformEvent{
		SystemIDs:  []string{"00000000-0000-0000-0000-000000000012"},
		RequestIDs: []string{"request-2"},
		AccountID:  rhAccountID,
		OrgID:      &orgID})
	assert.NoError(t, err)
	advisoryIDs := database.CheckAdvisoriesInDB(t, expectedAddedAdvisories)
	database.CheckAdvisoriesAccountDataNotified(t, rhAccountID, expectedAdvisoryIDs, true)

	assert.Equal(t, 1, len(mockWriter.Messages))

	var notificationSent ntf.Notification
	assert.Nil(t, json.Unmarshal(mockWriter.Messages[0].Value, &notificationSent))
	checkNotificationPayload(t, notificationSent, "RH-1", "enhancement", "adv-1-syn")
	checkNotificationPayload(t, notificationSent, "RH-2", "bugfix", "adv-2-syn")

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

	orgID := "1234567"
	notification := ntf.MakeNotification(inventoryID, strconv.Itoa(rhAccountID), orgID, NewAdvisoryEvent, events)
	assert.Equal(t, orgID, notification.OrgID)
	msg, err := mqueue.MessageFromJSON(inventoryID, notification)
	assert.Nil(t, err)
	assert.Equal(t, inventoryID, string(msg.Key))

	notificationJSON, err := json.Marshal(notification)
	assert.Nil(t, err)
	assert.Equal(t, notificationJSON, msg.Value)
}
