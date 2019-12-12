package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const id = "TEST-00000"

func deleteData(t *testing.T) {
	// Delete debug data from previous run
	assert.Nil(t, database.Db.Unscoped().Where("inventory_id = ?", id).Delete(&models.SystemPlatform{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("name = ?", id).Delete(&models.RhAccount{}).Error)
}

func TestGetOrCreateAccount(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	deleteData(t)

	acc, err := getOrCreateAccount(id)
	assert.Nil(t, err)
	acc2, err := getOrCreateAccount(id)
	assert.Nil(t, err)
	assert.Equal(t, acc, acc2)

	deleteData(t)
}

func TestUpdateSystemPlatform(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	deleteData(t)

	acc, err := getOrCreateAccount(id)
	assert.Nil(t, err)

	req := vmaas.UpdatesRequest{
		PackageList:    []string{"package0"},
		RepositoryList: []string{},
		ModulesList:    []vmaas.UpdatesRequestModulesList{},
		Releasever:     "7Server",
		Basearch:       "x86_64",
	}
	sys, err := updateSystemPlatform(id, acc, &req)
	assert.Nil(t, err)

	var system models.SystemPlatform
	assert.Nil(t, database.Db.Where("inventory_id = ?", id).Find(&system).Error)
	assert.Equal(t, system.InventoryID, id)
	assert.Equal(t, system.RhAccountID, acc)

	now := time.Now().Add(-time.Minute)

	assert.True(t, system.FirstReported.After(now), "First reported")
	assert.True(t, system.LastUpdated.After(now), "Last updated")
	assert.True(t, system.UnchangedSince.After(now), "Unchanged since")
	assert.True(t, system.LastUpload.After(now), "Last upload")
	// Last eval should be nil, system has not yet been evaluated
	assert.Nil(t, system.LastEvaluation)

	sys2, err := updateSystemPlatform(id, acc, &req)
	assert.Nil(t, err)

	assert.Equal(t, sys, sys2)

	deleteData(t)
}
