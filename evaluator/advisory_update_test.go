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
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()

	configure()
	loadCache()
	mockWriter := mqueue.MockKafkaWriter{}
	advisoryUpdatePublisher = &mockWriter

	enableAdvisoryUpdates = true
	defer func() { enableAdvisoryUpdates = false }()

	// Remove stale rows that may remain from previous test runs
	database.DeleteSystemAdvisories(t, testDBID, []int64{1, 2})
	database.DeleteAdvisoryAccountData(t, rhAccountID, []int64{1, 2})

	// Pair system with advisories before evaluation
	oldAdvisoryIDs := []int64{1, 3, 4}
	database.CreateSystemAdvisories(t, rhAccountID, testDBID, oldAdvisoryIDs)
	database.CreateAdvisoryAccountData(t, rhAccountID, oldAdvisoryIDs, 1)
	database.CheckCachesValid(t)

	// Run evaluation
	err := evaluateHandler(mqueue.PlatformEvent{
		SystemIDs:  []uuid.UUID{testInventoryID},
		RequestIDs: []string{"request-1"},
		OrgID:      &orgID,
		AccountID:  rhAccountID})
	assert.NoError(t, err)

	// Verify evaluated advisories exist in DB and get their IDs for cleanup
	evaluatedAdvisoryNames := []string{"RH-1", "RH-2", "RH-100"}
	evaluatedAdvisoryIDs := database.CheckAdvisoriesInDB(t, evaluatedAdvisoryNames)

	// Verify published Kafka message contains correct payload
	assert.GreaterOrEqual(t, len(mockWriter.Messages), 1)
	var event mqueue.AdvisoryUpdateEvent
	assert.NoError(t, sonic.Unmarshal(mockWriter.Messages[0].Value, &event))
	assert.Equal(t, rhAccountID, event.RhAccountID)
	assert.NotEmpty(t, event.AdvisoryIDs) // RH-100 gets a dynamic ID via lazy-save, so we only check delta is non-empty
	assert.False(t, event.ProducedAt.Time().IsZero())

	database.DeleteSystemAdvisories(t, testDBID, evaluatedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, evaluatedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, oldAdvisoryIDs)
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
