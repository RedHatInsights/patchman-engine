package mqueue

import (
	"app/base/types"
	"context"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
)

var (
	testWorkspaceIDs = []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"}
	testNow          = types.Rfc3339Timestamp(time.Now())
)

func TestAdvisoryUpdateEventMarshal(t *testing.T) {
	event := AdvisoryUpdateEvent{
		RhAccountID:  1,
		WorkspaceIDs: testWorkspaceIDs,
		AdvisoryIDs:  []int64{101, 202, 303},
		ProducedAt:   testNow,
	}

	data, err := sonic.Marshal(&event)
	assert.NoError(t, err)

	var parsed AdvisoryUpdateEvent
	err = sonic.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, event.RhAccountID, parsed.RhAccountID)
	assert.Equal(t, event.WorkspaceIDs, parsed.WorkspaceIDs)
	assert.Equal(t, event.AdvisoryIDs, parsed.AdvisoryIDs)
	assert.NotNil(t, parsed.ProducedAt)
}

func TestAdvisoryUpdateEventsWriteEvents(t *testing.T) {
	var writer Writer = &MockKafkaWriter{}

	events := AdvisoryUpdateEvents{
		{
			RhAccountID:  1,
			WorkspaceIDs: testWorkspaceIDs,
			AdvisoryIDs:  []int64{100, 200},
			ProducedAt:   testNow,
		},
		{
			RhAccountID:  2,
			WorkspaceIDs: testWorkspaceIDs,
			AdvisoryIDs:  []int64{300},
			ProducedAt:   testNow,
		},
	}

	err := SendMessages(context.Background(), writer, &events)
	assert.NoError(t, err)

	mockWriter := writer.(*MockKafkaWriter)
	assert.Equal(t, 2, len(mockWriter.Messages))

	var firstEvent AdvisoryUpdateEvent
	err = sonic.Unmarshal(mockWriter.Messages[0].Value, &firstEvent)
	assert.NoError(t, err)
	assert.Equal(t, events[0].RhAccountID, firstEvent.RhAccountID)
	assert.Equal(t, events[0].WorkspaceIDs, firstEvent.WorkspaceIDs)
	assert.Equal(t, events[0].AdvisoryIDs, firstEvent.AdvisoryIDs)
	assert.NotNil(t, firstEvent.ProducedAt)

	var secondEvent AdvisoryUpdateEvent
	err = sonic.Unmarshal(mockWriter.Messages[1].Value, &secondEvent)
	assert.NoError(t, err)
	assert.Equal(t, events[1].RhAccountID, secondEvent.RhAccountID)
	assert.Equal(t, events[1].WorkspaceIDs, secondEvent.WorkspaceIDs)
	assert.Equal(t, events[1].AdvisoryIDs, secondEvent.AdvisoryIDs)
	assert.NotNil(t, secondEvent.ProducedAt)
}
