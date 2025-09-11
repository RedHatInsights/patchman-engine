package kafka

import (
	"app/base/core"
	"app/base/mqueue"
	"app/base/utils"
	"app/manager/config"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
)

func testRecalcSystems(t *testing.T, accountID int, inventoryIDs []string) mqueue.PlatformEvent {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	config.EnableTemplateChangeEval = true

	writerMock := mqueue.MockKafkaWriter{}
	TryStartEvalQueue(mqueue.MockCreateKafkaWriter(&writerMock))
	inventoryAIDs := InventoryIDs2InventoryAIDs(accountID, inventoryIDs)
	RecalcSystems(inventoryAIDs)
	utils.AssertEqualWait(t, 1, func() (exp, act interface{}) {
		return 1, len(writerMock.Messages)
	})
	var event mqueue.PlatformEvent
	assert.Nil(t, sonic.Unmarshal(writerMock.Messages[0].Value, &event))
	return event
}

// Evaluate updated systems
func TestRecalcUpdatedSystems(t *testing.T) {
	inventoryIDs := []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000004"}
	event := testRecalcSystems(t, 1, inventoryIDs)
	assert.Equal(t, 2, len(event.SystemIDs))
	assert.Equal(t, 1, event.AccountID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", event.SystemIDs[0])
	assert.Equal(t, "00000000-0000-0000-0000-000000000004", event.SystemIDs[1])
}
