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

	"github.com/google/uuid"
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

	var inv models.SystemInventory
	assert.NotNil(t, staleDate)
	assert.NoError(t, database.DB.Where("stale = ?", false).Order("rh_account_id, id").First(&inv).Error)

	updateInventoryStaleFields(t, database.DB, &inv, &staleDate, &staleDate, false)

	nMarked, err := markSystemsStale(database.DB, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), nMarked)

	nMarked, err = markSystemsStale(database.DB, 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), nMarked)

	var updated models.SystemInventory
	assert.NoError(t, database.DB.First(&updated, "rh_account_id = ? AND id = ?", inv.RhAccountID, inv.ID).Error)
	assert.True(t, updated.Stale, "System should be marked stale")

	futureDate := time.Now().Add(24 * time.Hour)
	updateInventoryStaleFields(t, database.DB, &inv, &futureDate, &futureDate, true)

	nMarked, err = markSystemsStale(database.DB, 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), nMarked)

	assert.NoError(t, database.DB.First(&updated, "rh_account_id = ? AND id = ?", inv.RhAccountID, inv.ID).Error)
	assert.False(t, updated.Stale, "System should be marked not stale")

	// Cleanup fixture state
	updateInventoryStaleFields(t, database.DB, &inv, nil, nil, false)
}

func TestMarkSystemsStale(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	assert.NotNil(t, staleDate)
	inventories := loadAllSystemInventories(t, database.DB)
	for i := range inventories {
		assert.NotEqual(t, 0, inventories[i].ID)
		assert.False(t, inventories[i].Stale, "No systems should start stale")
		updateInventoryStaleFields(t, database.DB, &inventories[i], &staleDate, &staleDate, false)
	}

	nMarked, err := markSystemsStale(database.DB, 500)
	assert.NoError(t, err)
	assert.Equal(t, int64(18), nMarked)

	inventories = loadAllSystemInventories(t, database.DB)
	for i := range inventories {
		assert.True(t, inventories[i].Stale, "All systems should be marked stale")
		// Clean up fixture
		updateInventoryStaleFields(t, database.DB, &inventories[i], nil, nil, false)
	}
}

func TestMarkSystemsNotStale(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	assert.NotNil(t, staleDate)
	futureDate := time.Now().Add(24 * time.Hour)

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
		assert.True(t, inventories[i].Stale, "all systems should be stale before un-staling")
		updateInventoryStaleFields(t, database.DB, &inventories[i], &futureDate, &futureDate, true)
	}

	nMarked, err = markSystemsStale(database.DB, 500)
	assert.NoError(t, err)
	assert.Equal(t, int64(18), nMarked)

	inventories = loadAllSystemInventories(t, database.DB)
	for i := range inventories {
		assert.False(t, inventories[i].Stale, "all systems should be marked not stale")
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
			InventoryID:     uuid.MustParse(invID),
			RhAccountID:     1,
			DisplayName:     invID,
			Tags:            []byte("[]"),
			WorkspaceID:     database.TestWorkspace1UUID(),
			WorkspaceName:   database.TestWorkspace1NamePtr(),
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
}

func TestPruneDeletedSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	nToDelete := 4
	for i := 0; i < nToDelete; i++ {
		invID := fmt.Sprintf("00000000-0000-0000-0000-000000000de%d", i+1)
		assert.NoError(t, database.DB.Create(&models.DeletedSystem{
			InventoryID: uuid.MustParse(invID),
			WhenDeleted: staleDate,
		}).Error)
	}
	assert.NoError(t, database.DB.Create(&models.DeletedSystem{
		InventoryID: uuid.MustParse("00000000-0000-0000-0000-0000000000de"),
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
