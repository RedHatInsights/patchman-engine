package aggregator

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"sort"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
)

func toKafkaMessage(t *testing.T, event mqueue.AdvisoryUpdateEvent) mqueue.KafkaMessage {
	t.Helper()
	data, err := sonic.Marshal(event)
	assert.Nil(t, err)
	return mqueue.KafkaMessage{Value: data}
}

func TestGroupAdvisoryUpdatesSingleAccount(t *testing.T) {
	events := []mqueue.AdvisoryUpdateEvent{
		{RhAccountID: 1, AdvisoryIDs: []int64{1, 2}},
		{RhAccountID: 1, AdvisoryIDs: []int64{2, 3}},
	}
	grouped := groupAdvisoryUpdates(events)

	assert.Len(t, grouped, 1)
	ids := grouped[1]
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	assert.Equal(t, []int64{1, 2, 3}, ids)
}

func TestGroupAdvisoryUpdatesMultipleAccounts(t *testing.T) {
	events := []mqueue.AdvisoryUpdateEvent{
		{RhAccountID: 1, AdvisoryIDs: []int64{1, 2}},
		{RhAccountID: 2, AdvisoryIDs: []int64{5}},
		{RhAccountID: 1, AdvisoryIDs: []int64{3}},
	}
	grouped := groupAdvisoryUpdates(events)

	assert.Len(t, grouped, 2)

	ids1 := grouped[1]
	sort.Slice(ids1, func(i, j int) bool { return ids1[i] < ids1[j] })
	assert.Equal(t, []int64{1, 2, 3}, ids1)
	assert.Equal(t, []int64{int64(5)}, grouped[2])
}

func TestGroupAdvisoryUpdatesDeduplicates(t *testing.T) {
	events := []mqueue.AdvisoryUpdateEvent{
		{RhAccountID: 1, AdvisoryIDs: []int64{1, 1, 2}},
		{RhAccountID: 1, AdvisoryIDs: []int64{2, 2, 1}},
	}
	grouped := groupAdvisoryUpdates(events)

	ids := grouped[1]
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	assert.Equal(t, []int64{1, 2}, ids)
}

func TestBufferedEventsProcessedOnBatchThreshold(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	assert.Nil(t, database.TestDB().Exec("SELECT refresh_advisory_caches(NULL, 1)").Error)
	defer database.DeleteAccountAdvisoryByAccount(t, 1)

	msg := toKafkaMessage(t, mqueue.AdvisoryUpdateEvent{RhAccountID: 1, AdvisoryIDs: []int64{1, 2}})
	assert.Nil(t, advisoryUpdateHandler(base.Context, []mqueue.KafkaMessage{msg, msg, msg}))

	// Verify account_advisory was populated
	var count int64
	assert.Nil(t, database.TestDB().Table("account_advisory").
		Where("rh_account_id = 1 AND advisory_id IN (?)", []int64{1, 2}).
		Count(&count).Error)
	assert.True(t, count > 0, "account_advisory rows should be created after flush")
}
