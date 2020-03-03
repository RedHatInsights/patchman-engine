package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetLastRepobasedEvalTms(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	ts, err := getLastRepobasedEvalTms()
	assert.Nil(t, err)
	assert.Equal(t, "2018-04-04 23:23:45 +0000 UTC", ts.String())
}

func TestGetRepoBasedInventoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	repos := []string{"repo1", "repo3"}
	inventoryIDs, err := getRepoBasedInventoryIDs(repos)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(*inventoryIDs))
	assert.Equal(t, []string{"INV-1", "INV-5", "INV-6"}, *inventoryIDs)
}
