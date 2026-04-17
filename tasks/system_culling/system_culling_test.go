package system_culling

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/types"
	"app/base/utils"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var staleDate, _ = time.Parse(types.Rfc3339NoTz, "2006-01-02T15:04:05-07:00")

func loadAllSystemInventories(t *testing.T, db *gorm.DB) []models.SystemInventory {
	t.Helper()
	var rows []models.SystemInventory
	assert.NoError(t, db.Model(&models.SystemInventory{}).Order("rh_account_id, id").Find(&rows).Error)
	return rows
}

func loadFirstInstallableNonStaleInventory(t *testing.T, db *gorm.DB, rhAccountID int) models.SystemInventory {
	t.Helper()
	var inv models.SystemInventory
	err := db.Table("system_inventory AS si").
		Select("si.*").
		Joins("JOIN system_patch sp ON sp.system_id = si.id AND sp.rh_account_id = si.rh_account_id").
		Where("si.rh_account_id = ? AND si.stale = ? AND sp.installable_advisory_count_cache > ?",
			rhAccountID, false, 0).
		Order("si.id").
		First(&inv).Error
	assert.NoError(t, err)
	return inv
}

func updateInventoryStaleFields(t *testing.T, db *gorm.DB, inv *models.SystemInventory,
	staleTS, staleWarnTS *time.Time, stale bool,
) {
	t.Helper()
	assert.NoError(t, db.Model(&models.SystemInventory{}).
		Where("id = ? AND rh_account_id = ?", inv.ID, inv.RhAccountID).
		Updates(map[string]interface{}{
			"stale_timestamp":         staleTS,
			"stale_warning_timestamp": staleWarnTS,
			"stale":                   stale,
		}).Error)
}

func TestSingleSystemStale(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var oldAffected int
	var inv models.SystemInventory
	var accountData []models.AdvisoryAccountData

	database.DebugWithCachesCheck("stale-trigger", func() {
		assert.NotNil(t, staleDate)
		assert.NoError(t, database.DB.Find(&accountData, "systems_installable > 1 ").
			Order("systems_installable DESC").Error)
		inv = loadFirstInstallableNonStaleInventory(t, database.DB, accountData[0].RhAccountID)

		updateInventoryStaleFields(t, database.DB, &inv, &staleDate, &staleDate, inv.Stale)

		nMarked, err := markSystemsStale(database.DB, 0)
		assert.Nil(t, err)
		assert.Equal(t, int64(0), nMarked)

		nMarked, err = markSystemsStale(database.DB, 1)
		assert.Nil(t, err)
		assert.Equal(t, int64(1), nMarked)

		oldAffected = accountData[0].SystemsInstallable
		assert.NoError(t, database.DB.Find(&accountData, "rh_account_id = ? AND advisory_id = ?",
			accountData[0].RhAccountID, accountData[0].AdvisoryID).Error)

		assert.Equal(t, oldAffected-1, accountData[0].SystemsInstallable,
			"Systems affected should be decremented by one")
	})

	database.DebugWithCachesCheck("stale-trigger", func() {
		updateInventoryStaleFields(t, database.DB, &inv, nil, nil, false)
		assert.NoError(t, database.DB.Find(&accountData, "rh_account_id = ? AND advisory_id = ?",
			accountData[0].RhAccountID, accountData[0].AdvisoryID).Error)

		assert.Equal(t, oldAffected, accountData[0].SystemsInstallable,
			"Systems affected should be changed to match value at the start of the test case")
	})
}

// Test for making sure system culling works
func TestMarkSystemsStale(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	inventories := loadAllSystemInventories(t, database.DB)
	var accountData []models.AdvisoryAccountData
	assert.NotNil(t, staleDate)
	assert.NoError(t, database.DB.Find(&accountData).Error)
	for i := range inventories {
		assert.NotEqual(t, 0, inventories[i].ID)
		assert.Equal(t, false, inventories[i].Stale, "No systems should be stale")
		updateInventoryStaleFields(t, database.DB, &inventories[i], &staleDate, &staleDate, inventories[i].Stale)
	}

	assert.True(t, len(accountData) > 0, "We should have some systems affected by advisories")
	for _, a := range accountData {
		assert.True(t, a.SystemsInstallable+a.SystemsApplicable > 0, "We should have some systems affected")
	}
	nMarked, err := markSystemsStale(database.DB, 500)
	assert.Nil(t, err)
	assert.Equal(t, int64(18), nMarked)

	inventories = loadAllSystemInventories(t, database.DB)
	for i := range inventories {
		assert.Equal(t, true, inventories[i].Stale, "All systems should be stale")
		updateInventoryStaleFields(t, database.DB, &inventories[i], nil, nil, false)
	}

	assert.NoError(t, database.DB.Find(&accountData).Error)
	assert.True(t, len(accountData) > 0, "advisory_account_data should still exist after unstale")
	sumAffected := 0
	for _, a := range accountData {
		sumAffected += a.SystemsInstallable + a.SystemsApplicable
	}
	assert.True(t, sumAffected > 0,
		"after clearing stale, caches should show systems again (installable+applicable > 0)", sumAffected)
}

func TestMarkSystemsNotStale(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	// This test runs before TestMarkSystemsStale by name order; the DB fixture is not stale.
	// Match TestMarkSystemsStale setup so every host is stale, then verify clearing stale restores counts.
	var accountData []models.AdvisoryAccountData
	assert.NotNil(t, staleDate)

	inventories := loadAllSystemInventories(t, database.DB)
	for i := range inventories {
		assert.False(t, inventories[i].Stale, "fixture: systems should start non-stale")
		updateInventoryStaleFields(t, database.DB, &inventories[i], &staleDate, &staleDate, inventories[i].Stale)
	}
	nMarked, err := markSystemsStale(database.DB, 500)
	assert.NoError(t, err)
	assert.Equal(t, int64(18), nMarked)

	inventories = loadAllSystemInventories(t, database.DB)
	for i := range inventories {
		assert.True(t, inventories[i].Stale, "all systems should be stale after markSystemsStale")
		updateInventoryStaleFields(t, database.DB, &inventories[i], nil, nil, false)
	}

	assert.NoError(t, database.DB.Find(&accountData).Error)
	assert.True(t, len(accountData) > 0, "We should have some systems affected by advisories")
	for _, a := range accountData {
		assert.True(t, a.SystemsInstallable+a.SystemsApplicable > 0, "We should have some systems affected")
	}
}

func TestCullSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.TestLoadEnv("conf/test.env")
	core.SetupTestEnvironment()
	utils.TestLoadEnv("conf/vmaas_sync.env")

	nToDelete := 4
	for i := 0; i < nToDelete; i++ {
		invID := fmt.Sprintf("00000000-0000-0000-0000-000000000de%d", i+1)
		inv := models.SystemInventory{
			InventoryID:     invID,
			RhAccountID:     1,
			DisplayName:     invID,
			Tags:            []byte("[]"),
			CulledTimestamp: &staleDate,
		}
		assert.NoError(t, database.DB.Create(&inv).Error)
		assert.NoError(t, database.DB.Create(&models.SystemPatch{
			SystemID:    inv.ID,
			RhAccountID: 1,
		}).Error)
	}

	var cnt int64
	var cntAfter int64
	database.DebugWithCachesCheck("delete-culled", func() {
		assert.NoError(t, database.DB.Model(&models.SystemInventory{}).Count(&cnt).Error)
		// first batch
		nDeleted, err := deleteCulledSystems(database.DB, 3)
		assert.Nil(t, err)
		assert.Equal(t, int64(3), nDeleted)

		// second batch
		nDeleted, err = deleteCulledSystems(database.DB, 3)
		assert.Nil(t, err)
		assert.Equal(t, int64(1), nDeleted)

		assert.NoError(t, database.DB.Model(&models.SystemInventory{}).Count(&cntAfter).Error)
		assert.Equal(t, cnt-int64(nToDelete), cntAfter)
	})
}

func TestPruneDeletedSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	nToDelete := 4
	for i := 0; i < nToDelete; i++ {
		invID := fmt.Sprintf("00000000-0000-0000-0000-000000000de%d", i+1)
		assert.NoError(t, database.DB.Create(&models.DeletedSystem{
			InventoryID: invID,
			WhenDeleted: staleDate,
		}).Error)
	}
	assert.NoError(t, database.DB.Create(&models.DeletedSystem{
		InventoryID: "00000000-0000-0000-0000-000000000deff",
		WhenDeleted: time.Now(),
	}).Error)

	var cnt int64
	var cntAfter int64
	assert.NoError(t, database.DB.Model(&models.DeletedSystem{}).Count(&cnt).Error)
	assert.Equal(t, int64(nToDelete+1), cnt)

	nDeleted, err := pruneDeletedSystems(database.DB, 3)
	assert.Nil(t, err)
	assert.Equal(t, int64(3), nDeleted)

	// remove rest except last system (below threshold)
	nDeleted, err = pruneDeletedSystems(database.DB, 3)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), nDeleted)

	assert.NoError(t, database.DB.Model(&models.DeletedSystem{}).Count(&cntAfter).Error)
	assert.Equal(t, int64(1), cntAfter)

	// clean data from table
	assert.NoError(t, database.DB.Delete(&models.DeletedSystem{}, "1=1").Error)
}
