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
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

	database.DeleteSystemAdvisories(t, testDBID, expectedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, expectedAdvisoryIDs)
	database.CreateSystemAdvisories(t, rhAccountID, testDBID, oldSystemAdvisoryIDs)
	database.CreateAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs, 1)
	database.CheckCachesValid(t)
	database.CheckAdvisoriesAccountDataNotified(t, rhAccountID, oldSystemAdvisoryIDs, false)

	orgID := "1234567"
	// do evaluate the system
	data, err := sonic.Marshal(mqueue.PlatformEvent{
		SystemIDs:  []uuid.UUID{testInventoryID},
		RequestIDs: []string{"request-2"},
		AccountID:  rhAccountID,
		OrgID:      &orgID})
	assert.NoError(t, err)
	err = evaluateHandler(mqueue.KafkaMessage{Value: data})
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

	database.DeleteSystemAdvisories(t, testDBID, advisoryIDs)
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
	inv := &models.SystemInventory{
		InventoryID: testInventoryID,
		DisplayName: displayName,
	}
	tags := []ntf.SystemTag{{Key: "key", Namespace: "namespace", Value: "value"}}

	orgID := "1234567"
	url := fmt.Sprintf("https://localhost/insights/inventory/%s", testInventoryID.String())

	notification, err := ntf.MakeNotification(inv, tags, orgID, NewAdvisoryEvent, events)
	assert.Nil(t, err)
	assert.Equal(t, orgID, notification.OrgID)
	assert.Equal(t, url, notification.Context.HostURL)
	assert.Equal(t, testInventoryID, notification.Context.InventoryID)
	assert.Equal(t, displayName, notification.Context.DisplayName)
	assert.Equal(t, tags, notification.Context.Tags)

	msg, err := mqueue.MessageFromJSON(testInventoryID.String(), notification, nil)
	assert.Nil(t, err)
	assert.Equal(t, testInventoryID.String(), string(msg.Key))

	notificationJSON, err := sonic.Marshal(notification)
	assert.Nil(t, err)
	assert.Equal(t, notificationJSON, msg.Value)
}

func TestGetSystemTags(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	system := &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			ID:          1,
			RhAccountID: 1,
			InventoryID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			DisplayName: "display name",
		},
		Patch: models.SystemPatch{},
	}
	tags, err := getSystemTags(database.DB, system)
	expected := []ntf.SystemTag{
		{Key: "k1", Value: "val1", Namespace: "ns1"},
		{Key: "k2", Value: "val2", Namespace: "ns1"},
	}
	if assert.NoError(t, err) {
		assert.Equal(t, expected, tags)
	}
}

// TestAdvisoriesNotificationAlreadyNotified verifies that no Kafka message is sent when all
// advisories on the system have already been notified (notified IS NOT NULL). This is the
// exact scenario that caused the blank-email production bug introduced in RHINENG-21786.
func TestAdvisoriesNotificationAlreadyNotified(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	mockWriter := mqueue.MockKafkaWriter{}
	notificationsPublisher = &mockWriter

	advisoryIDs := []int64{1, 2}
	database.DeleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)

	now := time.Now()
	for _, id := range advisoryIDs {
		err := database.DB.Create(&models.AdvisoryAccountData{
			AdvisoryID:         id,
			RhAccountID:        rhAccountID,
			SystemsInstallable: 1,
			SystemsApplicable:  1,
			Notified:           &now,
		}).Error
		assert.NoError(t, err)
	}
	defer database.DeleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)

	system := &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			ID:          1,
			RhAccountID: rhAccountID,
			InventoryID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			DisplayName: "display name",
		},
		Patch: models.SystemPatch{},
	}
	newAdvs := SystemAdvisoryMap{
		"RH-1": {AdvisoryID: 1},
		"RH-2": {AdvisoryID: 2},
	}

	err := publishNewAdvisoriesNotification(database.DB, system, orgID, newAdvs)
	assert.NoError(t, err)
	assert.Empty(t, mockWriter.Messages, "no notification should be sent when all advisories are already notified")
}

// TestAdvisoriesNotificationEmptyAdvisoryMap verifies that no Kafka message is sent when
// publishNewAdvisoriesNotification is called with an empty SystemAdvisoryMap (no advisories
// on the system at all).
func TestAdvisoriesNotificationEmptyAdvisoryMap(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	mockWriter := mqueue.MockKafkaWriter{}
	notificationsPublisher = &mockWriter

	system := &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			ID:          1,
			RhAccountID: rhAccountID,
			InventoryID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			DisplayName: "display name",
		},
		Patch: models.SystemPatch{},
	}

	// An empty map means there is nothing to query — no messages should be produced regardless.
	publishNewAdvisoriesNotification(database.DB, system, orgID, SystemAdvisoryMap{}) //nolint:errcheck
	assert.Empty(t, mockWriter.Messages, "no notification should be sent when the advisory map is empty")
}

// TestGetUnnotifiedAdvisoriesReturnsEmpty documents the return-type contract of
// getUnnotifiedAdvisories: when all candidate advisories are already notified the function
// must return a non-nil empty slice (not nil). This prevents a future nil-vs-empty regression
// from silently bypassing the len == 0 guard in publishNewAdvisoriesNotification.
func TestGetUnnotifiedAdvisoriesReturnsEmpty(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	advisoryIDs := []int64{1, 2}
	database.DeleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)

	now := time.Now()
	for _, id := range advisoryIDs {
		err := database.DB.Create(&models.AdvisoryAccountData{
			AdvisoryID:         id,
			RhAccountID:        rhAccountID,
			SystemsInstallable: 1,
			SystemsApplicable:  1,
			Notified:           &now,
		}).Error
		assert.NoError(t, err)
	}
	defer database.DeleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)

	newAdvs := SystemAdvisoryMap{
		"RH-1": {AdvisoryID: 1},
		"RH-2": {AdvisoryID: 2},
	}

	result, err := getUnnotifiedAdvisories(database.DB, rhAccountID, newAdvs)
	assert.NoError(t, err)
	assert.NotNil(t, result, "result must be a non-nil slice so callers can use len() safely")
	assert.Empty(t, result, "no advisories should be returned when all are already notified")
}
