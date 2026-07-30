package aggregator

import (
	"app/base/core"
	"app/base/database"
	"app/base/mqueue"
	ntf "app/base/notification"
	"app/base/utils"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
)

func TestPublishNewAdvisoryNotificationSkipsAlreadyNotified(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	mockWriter := mqueue.MockKafkaWriter{}
	notificationsPublisher = &mockWriter
	enableNotifications = true
	defer func() {
		enableNotifications = false
		notificationsPublisher = nil
	}()

	testWorkspaceID := "00000000-0000-0000-0000-000000000001"
	advisoryIDs := []int64{1, 2}

	database.CreateAccountAdvisory(t, 1, testWorkspaceID, advisoryIDs, 1)
	// Mark them as already notified
	assert.Nil(t, database.DB.Table("account_advisory").
		Where("rh_account_id = 1 AND advisory_id IN (?)", advisoryIDs).
		Update("notified", "2026-01-01").Error)
	defer database.DeleteAccountAdvisoryByAccount(t, 1)

	err := publishNewAdvisoryNotification(1, advisoryIDs)
	assert.NoError(t, err)
	assert.Empty(t, mockWriter.Messages)
}

func TestPublishNewAdvisoryNotificationSuccess(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	mockWriter := mqueue.MockKafkaWriter{}
	notificationsPublisher = &mockWriter
	enableNotifications = true
	defer func() {
		enableNotifications = false
		notificationsPublisher = nil
	}()

	// Backfill to populate account_advisory from system_advisories
	assert.Nil(t, database.DB.Exec("SELECT backfill_account_advisory(1)").Error)
	defer database.DeleteAccountAdvisoryByAccount(t, 1)

	// Advisory IDs 1-8 exist for rh_account_id=1 in test data
	advisoryIDs := []int64{1, 2}

	err := publishNewAdvisoryNotification(1, advisoryIDs)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(mockWriter.Messages))
	assert.Equal(t, "org_1", string(mockWriter.Messages[0].Key))

	var notif ntf.Notification
	assert.Nil(t, sonic.Unmarshal(mockWriter.Messages[0].Value, &notif))
	assert.Equal(t, "org_1", notif.OrgID)
	assert.Nil(t, notif.Context)
	assert.NotEmpty(t, notif.Events)

	// Verify advisories were marked as notified (count varies by workspace)
	var count int64
	assert.Nil(t, database.DB.Table("account_advisory").
		Where("rh_account_id = 1 AND advisory_id IN (?) AND notified IS NOT NULL", advisoryIDs).
		Count(&count).Error)
	assert.True(t, count > 0)
}
