package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/core"
	"app/base/database"
	"app/base/mqueue"
	"app/base/types"
	"app/base/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetCurrentRepoBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	orgID := "org_1"
	inventoryAIDs, err := getCurrentRepoBasedInventoryIDs()
	assert.Nil(t, err)
	assert.Equal(t, []mqueue.InventoryAID{
		{InventoryID: "00000000-0000-0000-0000-000000000002", RhAccountID: 1, OrgID: &orgID},
		{InventoryID: "00000000-0000-0000-0000-000000000003", RhAccountID: 1, OrgID: &orgID}},
		inventoryAIDs)
	resetLastEvalTimestamp(t)
}

func TestGetLastRepobasedEvalTms(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	ts, err := database.GetTimestampKVValueStr(LastEvalRepoBased)
	assert.Nil(t, err)
	assert.Equal(t, "2018-04-04T23:23:45Z", *ts)
}

func TestUpdateRepoBaseEvalTimestamp(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	now := "2021-04-01T23:23:45Z"
	assert.Nil(t, database.UpdateTimestampKVValueStr(now, LastEvalRepoBased))

	ts, err := database.GetTimestampKVValueStr(LastEvalRepoBased)
	assert.Nil(t, err)
	assert.Equal(t, now, *ts)

	resetLastEvalTimestamp(t)
}

func TestGetRepoBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	orgID := "org_1"
	repos := []string{"repo1", "repo2"}
	inventoryAIDs, err := getRepoBasedInventoryIDs(repos)
	assert.Nil(t, err)
	assert.Equal(t, []mqueue.InventoryAID{
		{InventoryID: "00000000-0000-0000-0000-000000000002", RhAccountID: 1, OrgID: &orgID},
		{InventoryID: "00000000-0000-0000-0000-000000000003", RhAccountID: 1, OrgID: &orgID}},
		inventoryAIDs)
}

func TestGetRepoBasedInventoryIDsEmpty(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	repos := []string{}
	inventoryIDs, err := getRepoBasedInventoryIDs(repos)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(inventoryIDs))
}

func TestGetUpdatedRepos(t *testing.T) {
	core.SetupTestEnvironment()
	configure()

	modifiedSince := time.Now().Format(types.Rfc3339NoTz)
	redhat, thirdparty, err := getUpdatedRepos(time.Now(), &modifiedSince)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(redhat))
	assert.Equal(t, 0, len(thirdparty))
}

func resetLastEvalTimestamp(t *testing.T) {
	err := database.UpdateTimestampKVValueStr("2018-04-05T01:23:45+02:00", LastEvalRepoBased)
	assert.Nil(t, err)
}
