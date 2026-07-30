package system_advisories_0_recovery

import (
	"app/base/core"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNonStaleBucket0InventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	ids, err := getNonStaleBucket0InventoryIDs()
	require.NoError(t, err)

	var expected int64
	err = database.DB.Raw(`
		SELECT count(*)
		  FROM system_inventory si
		 WHERE si.stale = false
		   AND satisfies_hash_partition('system_advisories'::regclass, ?, ?, si.rh_account_id)
	`, systemAdvisoriesPartitions, systemAdvisoriesRemainder).Scan(&expected).Error
	require.NoError(t, err)
	assert.Equal(t, int(expected), len(ids))

	for _, row := range ids {
		var inBucket bool
		err = database.DB.Raw(
			`SELECT satisfies_hash_partition('system_advisories'::regclass, ?, ?, ?)`,
			systemAdvisoriesPartitions, systemAdvisoriesRemainder, row.RhAccountID,
		).Scan(&inBucket).Error
		require.NoError(t, err)
		assert.True(t, inBucket)
		assert.NotEqual(t, uuid.Nil, row.InventoryID)
	}
}

func TestPublishSetsSkipNotifications(t *testing.T) {
	orgID := "org_1"
	evals := mqueue.EvalDataSlice{
		{InventoryID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), RhAccountID: 1, OrgID: &orgID},
		{InventoryID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), RhAccountID: 1, OrgID: &orgID},
	}
	writer := &mqueue.MockKafkaWriter{}
	require.NoError(t, evals.WriteEventsSkipNotifications(context.Background(), writer, recoveryBatchSize))
	require.Len(t, writer.Messages, 1)

	var event mqueue.PlatformEvent
	require.NoError(t, sonic.Unmarshal(writer.Messages[0].Value, &event))
	assert.True(t, event.SkipNotifications)
	assert.Equal(t, 2, len(event.SystemIDs))
}
