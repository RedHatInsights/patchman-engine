package mqueue

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWriteEventsOfInventoryAccounts(t *testing.T) {
	const (
		acc  = 1
		inv2 = "00000000-0000-0000-0000-000000000002"
		inv3 = "00000000-0000-0000-0000-000000000003"
	)

	var writer Writer = &MockKafkaWriter{}

	orgID := "org_1"
	var invs EvalDataSlice = []EvalData{
		{InventoryID: inv2, RhAccountID: acc, OrgID: &orgID},
		{InventoryID: inv3, RhAccountID: acc, OrgID: &orgID}}

	assert.Nil(t, SendMessages(context.Background(), writer, &invs))

	mockWriter := writer.(*MockKafkaWriter)
	assert.True(t, len(mockWriter.Messages) > 0)

	var event PlatformEvent
	assert.Nil(t, json.Unmarshal(mockWriter.Messages[0].Value, &event))
	assert.Equal(t, orgID, event.GetOrgID())
	assert.Equal(t, acc, event.AccountID)
	assert.True(t, len(event.SystemIDs) == 2)
	assert.Equal(t, inv2, event.SystemIDs[0])
	assert.Equal(t, inv3, event.SystemIDs[1])
}
