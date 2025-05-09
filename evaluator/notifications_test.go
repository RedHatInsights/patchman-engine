package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	ntf "app/base/notification"
	"app/base/utils"
	"fmt"
	"testing"

	"github.com/bytedance/sonic"
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

	expectedAddedAdvisories := []string{"RH-1", "RH-2", "RH-100"}
	expectedAdvisoryIDs := []int64{1, 2}     // advisories expected to be paired to the system after evaluation
	oldSystemAdvisoryIDs := []int64{1, 3, 4} // old advisories paired with the system

	database.DeleteSystemAdvisories(t, systemID, expectedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, expectedAdvisoryIDs)
	database.CreateSystemAdvisories(t, rhAccountID, systemID, oldSystemAdvisoryIDs)
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
	assert.Nil(t, sonic.Unmarshal(mockWriter.Messages[0].Value, &notificationSent))
	checkNotificationPayload(t, notificationSent, "RH-1", "enhancement", "adv-1-syn")
	checkNotificationPayload(t, notificationSent, "RH-2", "bugfix", "adv-2-syn")

	events := notificationSent.Events
	// Assert is sorted ASC
	assert.True(t, events[0].Payload.(map[string]interface{})["advisory_name"].(string) <
		events[1].Payload.(map[string]interface{})["advisory_name"].(string))

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

	displayName := "display-name"
	system := &models.SystemPlatform{
		InventoryID: inventoryID,
		DisplayName: displayName,
	}
	tags := []ntf.SystemTag{{Key: "key", Namespace: "namespace", Value: "value"}}

	orgID := "1234567"
	url := fmt.Sprintf("https://localhost/insights/inventory/%s", inventoryID)

	notification, err := ntf.MakeNotification(system, tags, orgID, NewAdvisoryEvent, events)
	assert.Nil(t, err)
	assert.Equal(t, orgID, notification.OrgID)
	assert.Equal(t, url, notification.Context.HostURL)
	assert.Equal(t, inventoryID, notification.Context.InventoryID)
	assert.Equal(t, displayName, notification.Context.DisplayName)
	assert.Equal(t, tags, notification.Context.Tags)

	msg, err := mqueue.MessageFromJSON(inventoryID, notification)
	assert.Nil(t, err)
	assert.Equal(t, inventoryID, string(msg.Key))

	notificationJSON, err := sonic.Marshal(notification)
	assert.Nil(t, err)
	assert.Equal(t, notificationJSON, msg.Value)
}

func TestGetSystemTags(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	system := models.SystemPlatform{
		ID:          1,
		RhAccountID: 1,
		InventoryID: "00000000-0000-0000-0000-000000000001",
		DisplayName: "display name",
	}
	tags, err := getSystemTags(database.DB, &system)
	expected := []ntf.SystemTag{
		{Key: "k1", Value: "val1", Namespace: "ns1"},
		{Key: "k2", Value: "val2", Namespace: "ns1"},
	}
	if assert.NoError(t, err) {
		assert.Equal(t, expected, tags)
	}
}
