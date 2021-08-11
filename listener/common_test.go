package listener

import (
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"fmt"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const id = "99c0ffee-0000-0000-0000-0000c0ffee99"

func TestInit(t *testing.T) {
	utils.TestLoadEnv("conf/listener.env")
}

func deleteData(t *testing.T) {
	// Delete test data from previous run
	assert.Nil(t, database.Db.Unscoped().Exec("DELETE FROM advisory_account_data aad "+
		"USING rh_account ra WHERE ra.id = aad.rh_account_id AND ra.name = ?", id).Error)
	assert.Nil(t, database.Db.Unscoped().Where("first_reported > timestamp '2020-01-01'").
		Delete(&models.SystemAdvisories{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("name IN ('ER1', 'ER2', 'ER3')").
		Delete(&models.AdvisoryMetadata{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("repo_id NOT IN (1) OR system_id NOT IN (2, 3)").
		Delete(&models.SystemRepo{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("name NOT IN ('repo1', 'repo2', 'repo3', 'repo4')").
		Delete(&models.Repo{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("inventory_id = ?::uuid", id).Delete(&models.SystemPlatform{}).Error)
	assert.Nil(t, database.Db.Unscoped().Where("name = ?", id).Delete(&models.RhAccount{}).Error)
}

// nolint: unparam
func assertSystemInDB(t *testing.T, inventoryID string, rhAccountID *int, reporterID *int) {
	var system models.SystemPlatform
	assert.NoError(t, database.Db.Where("inventory_id = ?::uuid", inventoryID).Find(&system).Error)
	assert.Equal(t, system.InventoryID, inventoryID)

	var account models.RhAccount
	assert.NoError(t, database.Db.Where("id = ?", system.RhAccountID).Find(&account).Error)
	assert.Equal(t, inventoryID, account.Name)
	if rhAccountID != nil {
		assert.Equal(t, system.RhAccountID, *rhAccountID)
	}
	assert.Equal(t, system.ReporterID, reporterID)

	now := time.Now().Add(-time.Minute)
	assert.True(t, system.LastUpdated.After(now), "Last updated")
	assert.True(t, system.UnchangedSince.After(now), "Unchanged since")
	assert.True(t, system.LastUpload.After(now), "Last upload")
}

func assertSystemNotInDB(t *testing.T) {
	var systemCount int64
	assert.Nil(t, database.Db.Model(models.SystemPlatform{}).
		Where("inventory_id = ?::uuid", id).Count(&systemCount).Error)

	assert.Equal(t, int(systemCount), 0)
}

func getOrCreateTestAccount(t *testing.T) int {
	accountID, err := getOrCreateAccount(id)
	assert.Nil(t, err)
	return accountID
}

// nolint: unparam
func createTestUploadEvent(rhAccountID, inventoryID, reporter string, packages bool) HostEvent {
	ns := inventory.NewNullableString(utils.PtrString("insights"))
	v1 := inventory.NewNullableString(utils.PtrString("prod"))
	ev := HostEvent{
		Type:             "created",
		PlatformMetadata: nil,
		Host: Host{
			ID:       inventoryID,
			Account:  rhAccountID,
			Reporter: reporter,
			Tags: []inventory.StructuredTag{
				{
					Key:       utils.PtrString("env"),
					Namespace: *ns,
					Value:     *v1,
				}, {
					Key:       utils.PtrString("release"),
					Namespace: *ns,
					Value:     *v1,
				},
			},
		},
	}
	if packages {
		ev.Host.SystemProfile.SetInstalledPackages([]string{"kernel-54321.rhel8.x86_64"})
	}
	ev.Host.SystemProfile.SetDnfModules(
		[]inventory.SystemProfileSpecYamlDnfModule{{Name: utils.PtrString("modName"), Stream: utils.PtrString("modStream")}})
	ev.Host.SystemProfile.SetYumRepos(
		[]inventory.SystemProfileSpecYamlYumRepo{{Id: utils.PtrString("repo1"), Enabled: utils.PtrBool(true)}})

	return ev
}

func createTestDeleteEvent(inventoryID string) mqueue.PlatformEvent {
	typ := "delete"
	return mqueue.PlatformEvent{
		ID:   inventoryID,
		Type: &typ,
	}
}

func assertReposInDB(t *testing.T, repos []string) {
	var n []string
	err := database.Db.Model(&models.Repo{}).Where("name IN (?)", repos).Pluck("name", &n).Error
	fmt.Println(n)
	assert.Nil(t, err)
	assert.Equal(t, len(repos), len(n))
}

func assertSystemReposInDB(t *testing.T, systemID int, repos []string) {
	var c int64

	err := database.Db.Table("repo r").
		Joins("JOIN system_repo sr on sr.repo_id = r.id and sr.system_id = ? ", systemID).
		Where("r.name in (?)", repos).
		Count(&c).Error
	assert.NoError(t, err)
	assert.Equal(t, c, int64(len(repos)))
}
