package mqueue

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPlatformEventSkipNotificationsJSON(t *testing.T) {
	orgID := "org_1"
	event := PlatformEvent{
		AccountID:         1,
		OrgID:             &orgID,
		SkipNotifications: true,
	}
	data, err := sonic.Marshal(event)
	assert.NoError(t, err)

	var parsed PlatformEvent
	assert.NoError(t, sonic.Unmarshal(data, &parsed))
	assert.True(t, parsed.SkipNotifications)

	// omitempty: false must not appear in JSON; fresh unmarshal defaults to false
	event.SkipNotifications = false
	data, err = sonic.Marshal(event)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "skip_notifications")

	var parsedFalse PlatformEvent
	assert.NoError(t, sonic.Unmarshal(data, &parsedFalse))
	assert.False(t, parsedFalse.SkipNotifications)
}

func TestWriteEventsOfInventoryAccounts(t *testing.T) {
	var (
		acc  = 1
		inv2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
		inv3 = uuid.MustParse("00000000-0000-0000-0000-000000000003")
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
	assert.Nil(t, sonic.Unmarshal(mockWriter.Messages[0].Value, &event))
	assert.Equal(t, orgID, event.GetOrgID())
	assert.Equal(t, acc, event.AccountID)
	assert.True(t, len(event.SystemIDs) == 2)
	assert.Equal(t, inv2, event.SystemIDs[0])
	assert.Equal(t, inv3, event.SystemIDs[1])
	assert.False(t, event.SkipNotifications)
}

func TestWriteEventsSkipNotificationsChunking(t *testing.T) {
	acc := 7
	orgID := "org_recovery"
	invs := make(EvalDataSlice, 0, 501)
	for i := 0; i < 501; i++ {
		invs = append(invs, EvalData{
			InventoryID: uuid.New(),
			RhAccountID: acc,
			OrgID:       &orgID,
		})
	}

	writer := &MockKafkaWriter{}
	assert.NoError(t, invs.WriteEventsSkipNotifications(context.Background(), writer, 500))
	assert.Equal(t, 2, len(writer.Messages))

	var first, second PlatformEvent
	assert.NoError(t, sonic.Unmarshal(writer.Messages[0].Value, &first))
	assert.NoError(t, sonic.Unmarshal(writer.Messages[1].Value, &second))
	assert.True(t, first.SkipNotifications)
	assert.True(t, second.SkipNotifications)
	assert.Equal(t, 500, len(first.SystemIDs))
	assert.Equal(t, 1, len(second.SystemIDs))
	assert.Equal(t, acc, first.AccountID)
	assert.Equal(t, orgID, first.GetOrgID())
}
