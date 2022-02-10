package kafka

import (
	"app/base/core"
	"app/base/mqueue"
	"app/base/utils"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testEvaluateBaselineSystems(t *testing.T, baselineID *int, accountID int) mqueue.PlatformEvent {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	enableEvaluationRequests = true

	writerMock := mqueue.MockKafkaWriter{}
	TryStartEvalQueue(mqueue.MockCreateKafkaWriter(&writerMock))
	EvaluateBaselineSystems(baselineID, accountID)
	utils.AssertEqualWait(t, 1, func() (exp, act interface{}) {
		return 1, len(writerMock.Messages)
	})
	var event mqueue.PlatformEvent
	assert.Nil(t, json.Unmarshal(writerMock.Messages[0].Value, &event))
	return event
}

func TestEvaluateBaselineSystems1(t *testing.T) {
	event := testEvaluateBaselineSystems(t, utils.PtrInt(1), 1)
	assert.Equal(t, 2, len(event.SystemIDs))
	assert.Equal(t, 1, event.AccountID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", event.SystemIDs[0])
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", event.SystemIDs[1])
}

func TestEvaluateBaselineSystems2(t *testing.T) {
	event := testEvaluateBaselineSystems(t, utils.PtrInt(2), 1)
	assert.Equal(t, 1, len(event.SystemIDs))
	assert.Equal(t, 1, event.AccountID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", event.SystemIDs[0])
}

func TestEvaluateBaselineSystems3(t *testing.T) {
	event := testEvaluateBaselineSystems(t, nil, 1)
	assert.Equal(t, 5, len(event.SystemIDs))
	assert.Equal(t, 1, event.AccountID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000004", event.SystemIDs[0])
	assert.Equal(t, "00000000-0000-0000-0000-000000000005", event.SystemIDs[1])
	assert.Equal(t, "00000000-0000-0000-0000-000000000006", event.SystemIDs[2])
	assert.Equal(t, "00000000-0000-0000-0000-000000000007", event.SystemIDs[3])
	assert.Equal(t, "00000000-0000-0000-0000-000000000008", event.SystemIDs[4])
}
