package kafka

import (
	"app/base/core"
	"app/base/mqueue"
	"app/base/utils"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testEvaluateBaselineSystems(t *testing.T, baselineID *int, accountID int,
	configUpdated bool, inventoryIDs []string) mqueue.PlatformEvent {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	enableEvaluationRequests = true

	writerMock := mqueue.MockKafkaWriter{}
	TryStartEvalQueue(mqueue.MockCreateKafkaWriter(&writerMock))
	inventoryAIDs := GetInventoryIDsToEvaluate(baselineID, accountID, configUpdated, inventoryIDs)
	EvaluateBaselineSystems(inventoryAIDs)
	utils.AssertEqualWait(t, 1, func() (exp, act interface{}) {
		return 1, len(writerMock.Messages)
	})
	var event mqueue.PlatformEvent
	assert.Nil(t, json.Unmarshal(writerMock.Messages[0].Value, &event))
	return event
}

// Evaluate all baseline systems - config was updated, nothing added
func TestEvaluateBaselineSystemsDefault(t *testing.T) {
	event := testEvaluateBaselineSystems(t, utils.PtrInt(1), 1, true, nil)
	assert.Equal(t, 2, len(event.SystemIDs))
	assert.Equal(t, 1, event.AccountID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", event.SystemIDs[0])
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", event.SystemIDs[1])
}

// Evaluate just updated systems - config was not updated
func TestEvaluateBaselineUpdatedSystems(t *testing.T) {
	inventoryIDs := []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000004"}
	event := testEvaluateBaselineSystems(t, utils.PtrInt(1), 1, false, inventoryIDs)
	assert.Equal(t, 2, len(event.SystemIDs))
	assert.Equal(t, 1, event.AccountID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", event.SystemIDs[0])
	assert.Equal(t, "00000000-0000-0000-0000-000000000004", event.SystemIDs[1])
}

// Evaluate both all baseline systems and added ones - config updated, systems added
func TestEvaluateBaselineAllAndUpdatedSystems(t *testing.T) {
	inventoryIDs := []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000004"}
	event := testEvaluateBaselineSystems(t, utils.PtrInt(1), 1, true, inventoryIDs)
	assert.Equal(t, 3, len(event.SystemIDs))
	assert.Equal(t, 1, event.AccountID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", event.SystemIDs[0])
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", event.SystemIDs[1])
	assert.Equal(t, "00000000-0000-0000-0000-000000000004", event.SystemIDs[2])
}

// No systems needed to evaluate - e.g. just baseline name changed
func TestEvaluateBaselineNoSystems(t *testing.T) {
	inventoryAIDs := GetInventoryIDsToEvaluate(utils.PtrInt(1), 1, false, nil)
	assert.Equal(t, 0, len(inventoryAIDs))
}
