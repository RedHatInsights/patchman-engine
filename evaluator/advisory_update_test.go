package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetChangedAdvisoryIDs(t *testing.T) {
	advisories := extendedAdvisoryMap{
		"RH-1": {change: Add, SystemAdvisories: models.SystemAdvisories{AdvisoryID: 1}},
		"RH-2": {change: Update, SystemAdvisories: models.SystemAdvisories{AdvisoryID: 2}},
		"RH-3": {change: Remove, SystemAdvisories: models.SystemAdvisories{AdvisoryID: 3}},
		"RH-4": {change: Keep, SystemAdvisories: models.SystemAdvisories{AdvisoryID: 4}},
		"RH-5": {change: Keep, SystemAdvisories: models.SystemAdvisories{AdvisoryID: 5}},
	}

	ids := getChangedAdvisoryIDs(advisories)
	assert.ElementsMatch(t, []int64{1, 2, 3}, ids)
}

func TestCreateAdvisoryUpdateEvent(t *testing.T) {
	wsID := uuid.MustParse("d964b282-17f6-47ab-b596-a4a34d711f04")
	wsName := "test-workspace"
	system := &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			ID:            1,
			RhAccountID:   rhAccountID,
			InventoryID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			WorkspaceID:   &wsID,
			WorkspaceName: &wsName,
		},
		Patch: models.SystemPatch{},
	}

	changedAdvisoryIDs := []int64{1, 2}

	event := createAdvisoryUpdateEvent(system, changedAdvisoryIDs)
	assert.Equal(t, rhAccountID, event.RhAccountID)
	assert.Equal(t, wsID, event.WorkspaceID)
	assert.ElementsMatch(t, changedAdvisoryIDs, event.AdvisoryIDs)
	assert.False(t, event.ProducedAt.Time().IsZero())
}

func TestPublishAdvisoryUpdates(t *testing.T) {
	mockWriter := mqueue.MockKafkaWriter{}
	advisoryUpdatePublisher = &mockWriter

	wsID := uuid.MustParse("d964b282-17f6-47ab-b596-a4a34d711f04")
	wsName := "test-workspace"
	system := &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			ID:            1,
			RhAccountID:   rhAccountID,
			InventoryID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			WorkspaceID:   &wsID,
			WorkspaceName: &wsName,
		},
		Patch: models.SystemPatch{},
	}

	advisories := extendedAdvisoryMap{
		"RH-1": {change: Add, SystemAdvisories: models.SystemAdvisories{AdvisoryID: 1}},
		"RH-2": {change: Keep, SystemAdvisories: models.SystemAdvisories{AdvisoryID: 2}},
		"RH-3": {change: Remove, SystemAdvisories: models.SystemAdvisories{AdvisoryID: 3}},
	}

	err := publishAdvisoryUpdates(system, advisories)
	assert.NoError(t, err)
	assert.Len(t, mockWriter.Messages, 1)

	var event mqueue.AdvisoryUpdateEvent
	assert.NoError(t, sonic.Unmarshal(mockWriter.Messages[0].Value, &event))
	assert.Equal(t, rhAccountID, event.RhAccountID)
	assert.Equal(t, wsID, event.WorkspaceID)
	assert.ElementsMatch(t, []int64{1, 3}, event.AdvisoryIDs)
	assert.False(t, event.ProducedAt.Time().IsZero())
}

func TestPublishAdvisoryUpdatesNoDelta(t *testing.T) {
	mockWriter := mqueue.MockKafkaWriter{}
	advisoryUpdatePublisher = &mockWriter

	system := &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			ID:          1,
			RhAccountID: rhAccountID,
			InventoryID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		},
		Patch: models.SystemPatch{},
	}

	advisories := extendedAdvisoryMap{
		"RH-1": {change: Keep, SystemAdvisories: models.SystemAdvisories{AdvisoryID: 1}},
		"RH-2": {change: Keep, SystemAdvisories: models.SystemAdvisories{AdvisoryID: 2}},
	}

	err := publishAdvisoryUpdates(system, advisories)
	assert.NoError(t, err)
	assert.Empty(t, mockWriter.Messages)
}

func waitForConsumerGroup(t *testing.T, writer mqueue.Writer, received *mqueue.KafkaMessage) {
	t.Helper()
	err := writer.WriteMessages(t.Context(), mqueue.KafkaMessage{Value: []byte("probe")})
	assert.NoError(t, err, "failed to send probe message")
	utils.AssertEqualWait(t, 10, func() (exp, act interface{}) {
		return true, len(received.Value) > 0
	})
	*received = mqueue.KafkaMessage{}
}

func TestAdvisoryUpdateKafkaRoundTrip(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()

	configure()
	loadCache()

	topic := utils.CoreCfg.AdvisoryUpdateTopic
	if topic == "" {
		t.Skip("ADVISORY_UPDATE_TOPIC not configured")
	}

	advisoryUpdatePublisher = mqueue.NewKafkaWriterFromEnv(topic)
	reader := mqueue.NewKafkaReaderFromEnv(topic)
	defer func() {
		assert.NoError(t, reader.Close(), "failed to close Kafka reader")
	}()

	var received mqueue.KafkaMessage
	go reader.HandleMessages(t.Context(), func(m mqueue.KafkaMessage) error {
		received = m
		return nil
	})

	waitForConsumerGroup(t, advisoryUpdatePublisher, &received)

	// Remove stale rows from previous test runs
	database.DeleteSystemAdvisories(t, testDBID, []int64{1, 2})
	database.DeleteAdvisoryAccountData(t, rhAccountID, []int64{1, 2})

	// Pair system with advisories before evaluation
	oldAdvisoryIDs := []int64{1, 3, 4}
	database.CreateSystemAdvisories(t, rhAccountID, testDBID, oldAdvisoryIDs)
	database.CreateAdvisoryAccountData(t, rhAccountID, oldAdvisoryIDs, 1)
	database.CheckCachesValid(t)

	// Run evaluation
	data, err := sonic.Marshal(mqueue.PlatformEvent{
		SystemIDs:  []uuid.UUID{testInventoryID},
		RequestIDs: []string{"request-1"},
		OrgID:      &orgID,
		AccountID:  rhAccountID})
	assert.NoError(t, err)
	err = evaluateHandler(mqueue.KafkaMessage{Value: data})
	assert.NoError(t, err)

	utils.AssertEqualWait(t, 10, func() (exp, act interface{}) {
		return true, len(received.Value) > 0
	})
	var event mqueue.AdvisoryUpdateEvent
	assert.NoError(t, sonic.Unmarshal(received.Value, &event), "message should be valid AdvisoryUpdateEvent JSON")
	assert.Equal(t, rhAccountID, event.RhAccountID)
	assert.NotEqual(t, uuid.Nil, event.WorkspaceID)
	assert.NotEmpty(t, event.AdvisoryIDs)
	assert.False(t, event.ProducedAt.Time().IsZero())

	// Cleanup
	evaluatedAdvisoryNames := []string{"RH-1", "RH-2", "RH-100"}
	evaluatedAdvisoryIDs := database.CheckAdvisoriesInDB(t, evaluatedAdvisoryNames)
	database.DeleteSystemAdvisories(t, testDBID, evaluatedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, evaluatedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, oldAdvisoryIDs)
}
