package mqueue

import (
	"app/base"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var orgID string = "org_1"

func TestWriteEventsOfInventoryAccounts(t *testing.T) {
	var (
		acc  = 1
		inv2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
		inv3 = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	)

	var writer Writer = &MockKafkaWriter{}

	var invs EvalDataSlice = []EvalData{
		{InventoryID: inv2, RhAccountID: acc, OrgID: &orgID},
		{InventoryID: inv3, RhAccountID: acc, OrgID: &orgID}}

	assert.Nil(t, SendMessages(base.Context, writer, &invs))

	mockWriter := writer.(*MockKafkaWriter)
	assert.True(t, len(mockWriter.Messages) > 0)

	var event PlatformEvent
	assert.Nil(t, sonic.Unmarshal(mockWriter.Messages[0].Value, &event))
	assert.Equal(t, orgID, event.GetOrgID())
	assert.Equal(t, acc, event.AccountID)
	assert.True(t, len(event.SystemIDs) == 2)
	assert.Equal(t, inv2, event.SystemIDs[0])
	assert.Equal(t, inv3, event.SystemIDs[1])
}

func TestWriteEventsPreservesTraceparents(t *testing.T) {
	acc := 1
	inv2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	inv3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	writer := &MockKafkaWriter{}
	invs := EvalDataSlice{
		{InventoryID: inv2, RhAccountID: acc, OrgID: &orgID, RequestID: "r1",
			Traceparent: "00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01"},
		{InventoryID: inv3, RhAccountID: acc, OrgID: &orgID, RequestID: "r2",
			Traceparent: "00-cccccccccccccccccccccccccccccccc-dddddddddddddddd-01"},
	}
	assert.NoError(t, SendMessages(base.Context, writer, &invs))
	var event PlatformEvent
	assert.NoError(t, sonic.Unmarshal(writer.Messages[0].Value, &event))
	assert.Equal(t, invs[0].Traceparent, event.Traceparents[0])
	assert.Equal(t, invs[1].Traceparent, event.Traceparents[1])
	assert.Equal(t, []string{"r1", "r2"}, event.RequestIDs)
	assert.NotContains(t, string(writer.Messages[0].Value), "ingestion_origins")
	assert.NotContains(t, string(writer.Messages[0].Value), "started_at_ns")
}
