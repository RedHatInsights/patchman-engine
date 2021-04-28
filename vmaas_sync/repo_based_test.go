package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGetCurrentRepoBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	inventoryAIDs, err := getCurrentRepoBasedInventoryIDs()
	assert.Nil(t, err)
	assert.Equal(t, []inventoryAID{
		{"00000000-0000-0000-0000-000000000002", 1},
		{"00000000-0000-0000-0000-000000000003", 1}},
		inventoryAIDs)
	resetLastEvalTimestamp(t)
}

func TestGetLastRepobasedEvalTms(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	ts, err := database.GetTimestampKVValue(LastEvalRepoBased)
	assert.Nil(t, err)
	assert.Equal(t, "2018-04-04 23:23:45 +0000 UTC", ts.String())
}

func TestUpdateRepoBaseEvalTimestamp(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	database.UpdateTimestampKVValue(time.Now(), LastEvalRepoBased)

	ts, err := database.GetTimestampKVValue(LastEvalRepoBased)
	assert.Nil(t, err)
	assert.Equal(t, time.Now().Year(), ts.Year())

	resetLastEvalTimestamp(t)
}

func TestGetRepoBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	repos := []string{"repo1", "repo2"}
	inventoryAIDs, err := getRepoBasedInventoryIDs(repos)
	assert.Nil(t, err)
	assert.Equal(t, []inventoryAID{
		{"00000000-0000-0000-0000-000000000002", 1},
		{"00000000-0000-0000-0000-000000000003", 1}},
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

	modifiedSince := time.Now()
	repos, err := getUpdatedRepos(time.Now(), &modifiedSince, true)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(repos))
}

func resetLastEvalTimestamp(t *testing.T) {
	err := database.UpdateTimestampKVValueStr("2018-04-05T01:23:45+02:00", LastEvalRepoBased)
	assert.Nil(t, err)
}
