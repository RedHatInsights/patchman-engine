package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGetCurrentRepoBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	inventoryIDs, err := getCurrentRepoBasedInventoryIDs()
	assert.Nil(t, err)
	assert.Equal(t, []string{"INV-1", "INV-2", "INV-5", "INV-6"}, *inventoryIDs)
	resetLastEvalTimestamp(t)
}

func TestGetLastRepobasedEvalTms(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	ts, err := getLastRepobasedEvalTms()
	assert.Nil(t, err)
	assert.Equal(t, "2018-04-04 23:23:45 +0000 UTC", ts.String())
}

func TestUpdateRepoBaseEvalTimestamp(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	updateRepoBaseEvalTimestamp(time.Now())

	ts, err := getLastRepobasedEvalTms()
	assert.Nil(t, err)
	assert.Equal(t, time.Now().Year(), ts.Year())

	resetLastEvalTimestamp(t)
}

func TestGetRepoBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	repos := []string{"repo1", "repo3"}
	inventoryIDs, err := getRepoBasedInventoryIDs(&repos)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(*inventoryIDs))
	assert.Equal(t, []string{"INV-1", "INV-5", "INV-6"}, *inventoryIDs)
}

func TestGetUpdatedRepos(t *testing.T) {
	core.SetupTestEnvironment()
	configure()

	modifiedSince := time.Now()
	repos, err := getUpdatedRepos(&modifiedSince)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(*repos))
}

func resetLastEvalTimestamp(t *testing.T) {
	err := updateRepoBaseEvalTimestampStr("2018-04-05T01:23:45+02:00")
	assert.Nil(t, err)
}
