package vmaas_sync

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

var orgID1 = "org_1"

func TestGetCurrentRepoBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	Configure()

	inventoryAIDs, err := getCurrentRepoBasedInventoryIDs()
	assert.Nil(t, err)
	assert.Equal(t, []mqueue.EvalData{
		{InventoryID: "00000000-0000-0000-0000-000000000002", RhAccountID: 1, OrgID: &orgID1},
		{InventoryID: "00000000-0000-0000-0000-000000000003", RhAccountID: 1, OrgID: &orgID1},
		{InventoryID: "00000000-0000-0000-0000-000000000017", RhAccountID: 1, OrgID: &orgID1}},
		inventoryAIDs)
	resetLastEvalTimestamp(t)
}

func TestGetAllInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	Configure()

	inventoryAIDs, err := getAllInventoryIDs()
	systems := database.GetAllSystems(t)
	assert.Nil(t, err)
	assert.Equal(t, len(inventoryAIDs), len(systems))
	for i, inv := range inventoryAIDs {
		assert.NotNil(t, inv.OrgID)
		assert.Equal(t, systems[i].InventoryID, inv.InventoryID)
		assert.Equal(t, systems[i].RhAccountID, inv.RhAccountID)
	}
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
	assert.Nil(t, database.UpdateTimestampKVValueStr(LastEvalRepoBased, now))

	ts, err := database.GetTimestampKVValueStr(LastEvalRepoBased)
	assert.Nil(t, err)
	assert.Equal(t, now, *ts)

	resetLastEvalTimestamp(t)
}

func TestGetRepoOnlyBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	// some systems have repo, some have package
	// but none have both
	repos := []string{"repo3"}
	packages := []string{"curl", "bash"}
	inventoryAIDs, err := getRepoBasedInventoryIDs(repos, packages)
	assert.Nil(t, err)
	assert.Empty(t, inventoryAIDs)
}

func TestGetRepoPackageBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	// systems have both repo and package
	repos := []string{"not_exists_repo", "repo2"}
	packages := []string{"not_installed_pkg", "kernel"}
	inventoryAIDs, err := getRepoBasedInventoryIDs(repos, packages)
	assert.Nil(t, err)
	assert.Equal(t, []mqueue.EvalData{
		// "kernel" in "repo2"
		{InventoryID: "00000000-0000-0000-0000-000000000002", RhAccountID: 1, OrgID: &orgID1}},
		inventoryAIDs)

	repos = []string{"not_installed_pkg"}
	inventoryAIDs, err = getRepoBasedInventoryIDs(repos, nil)
	assert.Nil(t, err)
	assert.Len(t, inventoryAIDs, 0)
}

func TestGetRepoBasedInventoryIDsEmpty(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	repos := []string{}
	packages := []string{}
	inventoryIDs, err := getRepoBasedInventoryIDs(repos, packages)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(inventoryIDs))
}

func TestGetUpdatedRepos(t *testing.T) {
	core.SetupTestEnvironment()
	Configure()

	repos, err := getUpdatedRepos(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, 3, len(repos))
}

func TestGetUpdatedReposWithPackages(t *testing.T) {
	core.SetupTestEnvironment()
	Configure()

	modifiedSince := time.Now().Format(types.Rfc3339NoTz)
	repos, packages, _, err := getUpdatedReposWithPackages(time.Now(), &modifiedSince)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(repos))
	assert.Equal(t, 2, len(packages))
}

func resetLastEvalTimestamp(t *testing.T) {
	err := database.UpdateTimestampKVValueStr(LastEvalRepoBased, "2018-04-05T01:23:45+02:00")
	assert.Nil(t, err)
}
