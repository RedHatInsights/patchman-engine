package kafka

import (
	"app/base/core"
	"app/base/mqueue"
	"app/base/utils"
	"app/manager/config"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func testRecalcSystems(t *testing.T, accountID int, orgID string, inventoryIDs []uuid.UUID) mqueue.PlatformEvent {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	config.EnableTemplateChangeEval = true

	writerMock := mqueue.MockKafkaWriter{}
	TryStartEvalQueue(mqueue.MockCreateKafkaWriter(&writerMock))
	evaldataList := InventoryIDs2EvalData(accountID, orgID, inventoryIDs)
	RecalcSystems(evaldataList)
	utils.AssertEqualWait(t, 1, func() (exp, act interface{}) {
		return 1, len(writerMock.Messages)
	})
	var event mqueue.PlatformEvent
	assert.Nil(t, sonic.Unmarshal(writerMock.Messages[0].Value, &event))
	return event
}

// Evaluate updated systems
func TestRecalcUpdatedSystems(t *testing.T) {
	inventoryIDs := []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		uuid.MustParse("00000000-0000-0000-0000-000000000004"),
	}
	event := testRecalcSystems(t, 1, "org_1", inventoryIDs)
	assert.Equal(t, 2, len(event.SystemIDs))
	assert.Equal(t, 1, event.AccountID)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000001"), event.SystemIDs[0])
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000004"), event.SystemIDs[1])
}
