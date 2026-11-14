package aggregator

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/types"
	"app/base/utils"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	recountWorkspace = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherWorkspace   = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

func TestDedupeAdvisoryUpdatesKeepsYoungestProducedAt(t *testing.T) {
	older := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Minute)
	events := []mqueue.AdvisoryUpdateEvent{
		{RhAccountID: 2, WorkspaceID: recountWorkspace, AdvisoryIDs: []int64{5},
			ProducedAt: types.Rfc3339Timestamp(older)},
		{RhAccountID: 1, WorkspaceID: otherWorkspace, AdvisoryIDs: []int64{2, 1},
			ProducedAt: types.Rfc3339Timestamp(older)},
		{RhAccountID: 1, WorkspaceID: recountWorkspace, AdvisoryIDs: []int64{2, 1},
			ProducedAt: types.Rfc3339Timestamp(older)},
		{RhAccountID: 1, WorkspaceID: recountWorkspace, AdvisoryIDs: []int64{1},
			ProducedAt: types.Rfc3339Timestamp(newer)},
	}

	got := groupAdvisoryUpdates(events)

	assert.Equal(t, []advisoryRecount{
		{RhAccountID: 1, WorkspaceID: recountWorkspace, AdvisoryID: 1, ProducedAt: newer},
		{RhAccountID: 1, WorkspaceID: recountWorkspace, AdvisoryID: 2, ProducedAt: older},
		{RhAccountID: 1, WorkspaceID: otherWorkspace, AdvisoryID: 1, ProducedAt: older},
		{RhAccountID: 1, WorkspaceID: otherWorkspace, AdvisoryID: 2, ProducedAt: older},
		{RhAccountID: 2, WorkspaceID: recountWorkspace, AdvisoryID: 5, ProducedAt: older},
	}, got)
}

func toKafkaMessage(t *testing.T, event mqueue.AdvisoryUpdateEvent) mqueue.KafkaMessage {
	t.Helper()
	data, err := sonic.Marshal(event)
	assert.Nil(t, err)
	return mqueue.KafkaMessage{Value: data}
}

func TestBufferedEventsProcessedOnBatchThreshold(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	assert.Nil(t, database.TestDB().Exec("SELECT refresh_advisory_caches(NULL, 1)").Error)
	defer database.DeleteAccountAdvisoryByAccount(t, 1)

	assert.Nil(t, database.TestDB().
		Where("rh_account_id = 1 AND workspace_id = ? AND advisory_id = 1", otherWorkspace).
		Delete(&models.AccountAdvisory{}).Error)
	assert.Nil(t, database.TestDB().Create(&models.AccountAdvisory{
		AdvisoryID: 1, RhAccountID: 1, WorkspaceID: otherWorkspace,
		SystemsInstallable: 77, SystemsApplicable: 77,
	}).Error)

	msg := toKafkaMessage(t, mqueue.AdvisoryUpdateEvent{
		RhAccountID: 1, WorkspaceID: recountWorkspace, AdvisoryIDs: []int64{2, 1},
		ProducedAt: types.Rfc3339Timestamp(time.Now().Add(time.Hour)),
	})
	assert.Nil(t, advisoryUpdateHandler(base.Context, []mqueue.KafkaMessage{msg, msg, msg}))

	var count int64
	assert.Nil(t, database.TestDB().Table("account_advisory").
		Where("rh_account_id = 1 AND workspace_id = ? AND advisory_id IN (?)", recountWorkspace, []int64{1, 2}).
		Count(&count).Error)
	assert.Equal(t, int64(2), count)

	installable := database.PluckInt(database.TestDB().Table("account_advisory").
		Where("rh_account_id = 1 AND workspace_id = ? AND advisory_id = 1", otherWorkspace),
		"systems_installable")
	assert.Equal(t, 77, installable)
}

func TestRecountUpdatesCountsAndDeletesEmpty(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	database.DeleteAccountAdvisory(t, 1, recountWorkspace.String(), []int64{1, 9})
	t.Cleanup(func() {
		database.DeleteAccountAdvisory(t, 1, recountWorkspace.String(), []int64{1, 9})
	})

	assert.Nil(t, database.TestDB().Create(&models.AccountAdvisory{
		AdvisoryID: 1, RhAccountID: 1, WorkspaceID: recountWorkspace,
		SystemsInstallable: 99, SystemsApplicable: 99,
	}).Error)

	assert.Nil(t, recountAdvisory(base.Context, 1, recountWorkspace, 1, time.Now().Add(time.Hour)))

	var expected int
	assert.Nil(t, database.TestDB().Raw(`
		SELECT count(sa.*) FILTER (WHERE sa.status_id = 0)
		  FROM system_advisories sa
		  JOIN system_inventory si
		    ON sa.rh_account_id = si.rh_account_id AND sa.system_id = si.id
		  JOIN system_patch sp
		    ON si.id = sp.system_id AND sp.rh_account_id = si.rh_account_id
		 WHERE sp.last_evaluation IS NOT NULL
		   AND si.stale = FALSE
		   AND sa.rh_account_id = ?
		   AND si.workspace_id = ?
		   AND sa.advisory_id = ?`,
		1, recountWorkspace, 1).Scan(&expected).Error)

	got := database.PluckInt(database.TestDB().Table("account_advisory").
		Where("rh_account_id = 1 AND workspace_id = ? AND advisory_id = 1", recountWorkspace),
		"systems_installable")
	assert.Equal(t, expected, got)
	assert.NotEqual(t, 99, got)

	assert.Nil(t, database.TestDB().Create(&models.AccountAdvisory{
		AdvisoryID: 9, RhAccountID: 1, WorkspaceID: recountWorkspace,
		SystemsInstallable: 1, SystemsApplicable: 1,
	}).Error)
	assert.Nil(t, recountAdvisory(base.Context, 1, recountWorkspace, 9, time.Now().Add(time.Hour)))

	var left int64
	assert.Nil(t, database.TestDB().Table("account_advisory").
		Where("rh_account_id = 1 AND workspace_id = ? AND advisory_id = 9", recountWorkspace).
		Count(&left).Error)
	assert.Equal(t, int64(0), left)
}

func setAdvisoryUpdated(t *testing.T, advisoryID int64, installable int, updated time.Time) {
	t.Helper()
	database.DeleteAccountAdvisory(t, 1, recountWorkspace.String(), []int64{advisoryID})
	assert.Nil(t, database.TestDB().Create(&models.AccountAdvisory{
		AdvisoryID: advisoryID, RhAccountID: 1, WorkspaceID: recountWorkspace,
		SystemsInstallable: installable, SystemsApplicable: installable,
	}).Error)
	assert.Nil(t, database.TestDB().Model(&models.AccountAdvisory{}).
		Where("rh_account_id = 1 AND workspace_id = ? AND advisory_id = ?", recountWorkspace, advisoryID).
		UpdateColumns(map[string]interface{}{
			"systems_installable": installable,
			"systems_applicable":  installable,
			"updated":             updated,
		}).Error)
}

func TestRecountSkipsWhenUpdatedIsYoungerThanProducedAt(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	t.Cleanup(func() { database.DeleteAccountAdvisory(t, 1, recountWorkspace.String(), []int64{1}) })

	producedAt := time.Now().UTC().Truncate(time.Microsecond)
	setAdvisoryUpdated(t, 1, 99, producedAt.Add(time.Hour))

	assert.Nil(t, recountAdvisory(base.Context, 1, recountWorkspace, 1, producedAt))

	got := database.PluckInt(database.TestDB().Table("account_advisory").
		Where("rh_account_id = 1 AND workspace_id = ? AND advisory_id = 1", recountWorkspace),
		"systems_installable")
	assert.Equal(t, 99, got)
}

func TestRecountRunsWhenUpdatedIsOlderOrEqual(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	t.Cleanup(func() { database.DeleteAccountAdvisory(t, 1, recountWorkspace.String(), []int64{1}) })

	producedAt := time.Now().UTC().Truncate(time.Microsecond)
	setAdvisoryUpdated(t, 1, 99, producedAt.Add(-time.Hour))
	assert.Nil(t, recountAdvisory(base.Context, 1, recountWorkspace, 1, producedAt))
	got := database.PluckInt(database.TestDB().Table("account_advisory").
		Where("rh_account_id = 1 AND workspace_id = ? AND advisory_id = 1", recountWorkspace),
		"systems_installable")
	assert.NotEqual(t, 99, got)

	setAdvisoryUpdated(t, 1, 99, producedAt)
	assert.Nil(t, recountAdvisory(base.Context, 1, recountWorkspace, 1, producedAt))
	got = database.PluckInt(database.TestDB().Table("account_advisory").
		Where("rh_account_id = 1 AND workspace_id = ? AND advisory_id = 1", recountWorkspace),
		"systems_installable")
	assert.NotEqual(t, 99, got)
}
