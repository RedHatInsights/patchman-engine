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

	repos := []string{"repo1", "repo2"}
	inventoryAIDs, err := getRepoBasedInventoryIDs(nil, repos)
	assert.Nil(t, err)
	assert.Equal(t, []mqueue.EvalData{
		{InventoryID: "00000000-0000-0000-0000-000000000002", RhAccountID: 1, OrgID: &orgID1},
		{InventoryID: "00000000-0000-0000-0000-000000000003", RhAccountID: 1, OrgID: &orgID1},
		{InventoryID: "00000000-0000-0000-0000-000000000017", RhAccountID: 1, OrgID: &orgID1}},
		inventoryAIDs)
}

func TestGetRepoPackageBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	repos := [][]string{{"repo1", "not_installed_pkg"}, {"repo2", "not_installed_pkg"}, {"repo2", "kernel"}}
	inventoryAIDs, err := getRepoBasedInventoryIDs(repos, nil)
	assert.Nil(t, err)
	assert.Equal(t, []mqueue.EvalData{
		// "kernel" in "repo2"
		{InventoryID: "00000000-0000-0000-0000-000000000002", RhAccountID: 1, OrgID: &orgID1}},
		// 00000000-0000-0000-0000-000000000017 does not have "not_installed_pkg" in "repo1"
		inventoryAIDs)

	repos = [][]string{{"repo1", "not_installed_pkg"}, {"repo2", "not_installed_pkg"}}
	inventoryAIDs, err = getRepoBasedInventoryIDs(repos, nil)
	assert.Nil(t, err)
	assert.Len(t, inventoryAIDs, 0)
}

func TestGetRepoBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	repos := []string{"repo1"}
	repoPackages := [][]string{{"repo1", "not_installed_pkg"}, {"repo2", "not_installed_pkg"}, {"repo2", "kernel"}}
	inventoryAIDs, err := getRepoBasedInventoryIDs(repoPackages, repos)
	assert.Nil(t, err)
	assert.Equal(t, []mqueue.EvalData{
		// from repoPackages
		{InventoryID: "00000000-0000-0000-0000-000000000002", RhAccountID: 1, OrgID: &orgID1},
		// systems added from repos
		{InventoryID: "00000000-0000-0000-0000-000000000003", RhAccountID: 1, OrgID: &orgID1},
		{InventoryID: "00000000-0000-0000-0000-000000000017", RhAccountID: 1, OrgID: &orgID1}},
		inventoryAIDs)
}

func TestGetRepoBasedInventoryIDsEmpty(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	repos := []string{}
	repoPackages := [][]string{}
	inventoryIDs, err := getRepoBasedInventoryIDs(repoPackages, repos)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(inventoryIDs))
}

func TestGetUpdatedRepos(t *testing.T) {
	core.SetupTestEnvironment()
	Configure()

	modifiedSince := time.Now().Format(types.Rfc3339NoTz)
	thirdParty := true
	repoPackages, repoNoPackages, _, err := getUpdatedRepos(time.Now(), &modifiedSince, &thirdParty)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(repoPackages[0]))
	assert.Equal(t, 2, len(repoNoPackages))
}

func resetLastEvalTimestamp(t *testing.T) {
	err := database.UpdateTimestampKVValueStr(LastEvalRepoBased, "2018-04-05T01:23:45+02:00")
	assert.Nil(t, err)
}
