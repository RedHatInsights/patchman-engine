package listener

import (
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const id = "TEST-00000"

func deleteData(t *testing.T) {
	// Delete test data from previous run
	assert.Nil(t, database.Db.Unscoped().Exec("DELETE FROM advisory_account_data aad "+
		"USING rh_account ra WHERE ra.id = aad.rh_account_id AND ra.name = ?", id).Error)
	assert.Nil(t, database.Db.Unscoped().Where("first_reported > timestamp '2020-01-01'").
		Delete(&models.SystemAdvisories{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("name IN ('ER1', 'ER2', 'ER3')").
		Delete(&models.AdvisoryMetadata{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("inventory_id = ?", id).Delete(&models.SystemPlatform{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("name = ?", id).Delete(&models.RhAccount{}).Error)
}

// nolint:unparam
func assertSystemInDb(t *testing.T, inventoryID string, rhAccountID *int) {
	var system models.SystemPlatform
	assert.NoError(t, database.Db.Where("inventory_id = ?", inventoryID).Find(&system).Error)
	assert.Equal(t, system.InventoryID, inventoryID)

	var account models.RhAccount
	assert.NoError(t, database.Db.Where("id = ?", system.RhAccountID).Find(&account).Error)
	assert.Equal(t, inventoryID, account.Name)
	if rhAccountID != nil {
		assert.Equal(t, system.RhAccountID, *rhAccountID)
	}

	now := time.Now().Add(-time.Minute)
	assert.True(t, system.FirstReported.After(now), "First reported")
	assert.True(t, system.LastUpdated.After(now), "Last updated")
	assert.True(t, system.UnchangedSince.After(now), "Unchanged since")
	assert.True(t, system.LastUpload.After(now), "Last upload")
}

func assertSystemNotInDb(t *testing.T) {
	var systemCount int
	assert.Nil(t, database.Db.Model(models.SystemPlatform{}).
		Where("inventory_id = ?", id).Count(&systemCount).Error)

	assert.Equal(t, systemCount, 0)
}

func getOrCreateTestAccount(t *testing.T) int {
	accountID, err := getOrCreateAccount(id)
	assert.Nil(t, err)
	return accountID
}

//nolint directives
func createTestUploadEvent(inventoryID string, packages bool) HostEgressEvent {
	ev := HostEgressEvent{
		Type:             "created",
		PlatformMetadata: nil,
		Host:             Host{ID: inventoryID, Account: inventoryID},
	}
	if packages {
		ev.Host.SystemProfile.InstalledPackages = []string{"kernel-54321.rhel8.x86_64"}
	}
	ev.Host.SystemProfile.DnfModules = []inventory.DnfModule{{Name: "modName", Stream: "modStream"}}
	ev.Host.SystemProfile.YumRepos = []inventory.YumRepo{{Name: "repoName", Enabled: true}}
	return ev
}

func createTestDeleteEvent(inventoryID string) mqueue.PlatformEvent {
	typ := "delete"
	return mqueue.PlatformEvent{
		ID:   inventoryID,
		Type: &typ,
	}
}
