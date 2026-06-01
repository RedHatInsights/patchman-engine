package workspace_backfill

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type workspaceRow struct {
	WorkspaceID   *string    `gorm:"column:workspace_id"`
	WorkspaceName *string    `gorm:"column:workspace_name"`
	LastUpdated   *time.Time `gorm:"column:last_updated"`
}

//nolint:funlen
func TestBackfillBatch(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	lastUpdated := time.Date(2020, 1, 15, 12, 0, 0, 0, time.UTC)
	var systemIDs []int64

	for i := 0; i < 3; i++ {
		invID := fmt.Sprintf("00000000-0000-4000-8000-00000000%04x", i)
		inv := models.SystemInventory{
			InventoryID: uuid.MustParse(invID),
			RhAccountID: 1,
			DisplayName: invID,
			Tags:        []byte("[]"),
			Workspaces:  database.TestWorkspacesGroup1(),
		}
		require.NoError(t, database.DB.Create(&inv).Error)
		systemIDs = append(systemIDs, inv.ID)

		require.NoError(t, database.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(setReplicaRoleSQL).Error; err != nil {
				return err
			}
			return tx.Exec(
				`UPDATE system_inventory
				 SET workspace_id = NULL, workspace_name = NULL, last_updated = ?
				 WHERE rh_account_id = ? AND id = ?`,
				lastUpdated, 1, inv.ID,
			).Error
		}))
	}

	for _, systemID := range systemIDs {
		var row workspaceRow
		require.NoError(t, database.DB.Table("system_inventory").
			Select("workspace_id, workspace_name, last_updated").
			Where("rh_account_id = ? AND id = ?", 1, systemID).
			Scan(&row).Error)
		assert.Nil(t, row.WorkspaceID)
		assert.Nil(t, row.WorkspaceName)
		require.NotNil(t, row.LastUpdated)
		assert.True(t, row.LastUpdated.Equal(lastUpdated))
	}

	rows, err := backfillBatch(1, 1000)
	require.NoError(t, err)
	assert.Equal(t, int64(3), rows)

	for _, systemID := range systemIDs {
		var row workspaceRow
		require.NoError(t, database.DB.Table("system_inventory").
			Select("workspace_id, workspace_name, last_updated").
			Where("rh_account_id = ? AND id = ?", 1, systemID).
			Scan(&row).Error)
		require.NotNil(t, row.WorkspaceID)
		assert.Equal(t, database.TestWorkspace1ID, *row.WorkspaceID)
		require.NotNil(t, row.WorkspaceName)
		assert.Equal(t, "group1", *row.WorkspaceName)
		require.NotNil(t, row.LastUpdated)
		assert.True(t, row.LastUpdated.Equal(lastUpdated), "last_updated must be preserved")
	}

	rows, err = backfillBatch(1, 1000)
	require.NoError(t, err)
	assert.Equal(t, int64(0), rows)

	for i := range systemIDs {
		invID := fmt.Sprintf("00000000-0000-4000-8000-00000000%04x", i)
		require.NoError(t, database.DB.Exec("SELECT delete_system(?::uuid)", invID).Error)
	}
}
