package workspace_backfill

import (
	"app/base/core"
	"app/base/utils"
	"app/tasks"
	"time"

	"gorm.io/gorm"
)

const setReplicaRoleSQL = `SET LOCAL session_replication_role = replica`

const workspacePendingPredicate = `
workspace_id IS NULL
  AND workspaces IS NOT NULL
  AND jsonb_typeof(workspaces) = 'array'
  AND jsonb_array_length(workspaces) > 0
  AND workspaces->0->>'id' IS NOT NULL
  AND workspaces->0->>'name' IS NOT NULL
  AND NOT empty(workspaces->0->>'name')`

const validWorkspacesJSONPredicate = `
jsonb_typeof(workspaces) = 'array'
  AND jsonb_array_length(workspaces) > 0
  AND workspaces->0->>'id' IS NOT NULL
  AND workspaces->0->>'name' IS NOT NULL
  AND NOT empty(workspaces->0->>'name')`

const backfillUpdateSQL = `
UPDATE system_inventory si
SET workspace_id   = (si.workspaces->0->>'id')::uuid,
    workspace_name = si.workspaces->0->>'name'
FROM (
    SELECT rh_account_id, id
    FROM system_inventory
    WHERE rh_account_id = ?
      AND ` + workspacePendingPredicate + `
    ORDER BY id
    LIMIT ?
) batch
WHERE si.rh_account_id = batch.rh_account_id
  AND si.id = batch.id`

const pendingAccountsSQL = `
SELECT rh_account_id
FROM system_inventory
WHERE ` + workspacePendingPredicate + `
GROUP BY rh_account_id
ORDER BY hash_partition_id(rh_account_id, 128), rh_account_id`

const pendingRowsSQL = workspacePendingPredicate

const invalidPendingRowsSQL = `
workspace_id IS NULL
  AND workspaces IS NOT NULL
  AND NOT (` + validWorkspacesJSONPredicate + `)`

func configure() {
	// Admin DB user (same as migrations): UPDATE on partitions and session_replication_role = replica.
	core.ConfigureAdminApp()
}

// RunWorkspaceBackfill runs the batched workspace_id / workspace_name backfill job.
func RunWorkspaceBackfill() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	configure()

	nUpdated, complete, err := runWorkspaceBackfill()
	if err != nil {
		utils.LogError("err", err.Error(), "Workspace backfill")
		return
	}
	if err := Metrics().Add(); err != nil {
		utils.LogInfo("err", err, "Could not push to pushgateway")
	}
	if complete {
		utils.LogInfo("nUpdated", nUpdated, "Workspace backfill complete")
	} else {
		utils.LogInfo("nUpdated", nUpdated, "Workspace backfill paused (per-run limit); more rows remain")
	}
}

func runWorkspaceBackfill() (nUpdated int64, complete bool, err error) {
	if err := logPendingStats(); err != nil {
		return 0, false, err
	}

	accounts, err := loadPendingAccounts()
	if err != nil {
		return 0, false, err
	}
	if len(accounts) == 0 {
		return 0, true, nil
	}

	utils.LogInfo("accounts", len(accounts), "Starting workspace backfill")

	maxRows := int64(tasks.WorkspaceBackfillMaxRowsPerRun)
	var total int64

	for i, rhAccountID := range accounts {
		rows, hitLimit, batchErr := processAccountBatches(i, rhAccountID, maxRows, total)
		total += rows
		if batchErr != nil {
			continue
		}
		if hitLimit {
			return total, false, nil
		}
	}

	pending, err := countPending()
	if err != nil {
		return total, false, err
	}
	return total, pending == 0, nil
}

//nolint:lll
func processAccountBatches(idx, rhAccountID int, maxRows, totalSoFar int64) (rowsUpdated int64, hitLimit bool, err error) {
	for totalSoFar+rowsUpdated < maxRows {
		remaining := maxRows - (totalSoFar + rowsUpdated)
		batchLimit := tasks.WorkspaceBackfillBatchSize
		if int64(batchLimit) > remaining {
			batchLimit = int(remaining)
		}

		rows, batchErr := backfillBatch(rhAccountID, batchLimit)
		if batchErr != nil {
			utils.LogWarn("rhAccountID", rhAccountID, "err", batchErr.Error(), "Workspace backfill batch failed")
			backfillErrorsCnt.Inc()
			return rowsUpdated, false, batchErr
		}
		if rows == 0 {
			return rowsUpdated, false, nil
		}

		rowsUpdated += rows
		backfillRowsCnt.Add(float64(rows))
		backfillBatchesCnt.Inc()
		utils.LogInfo("i", idx, "rhAccountID", rhAccountID, "nRows", rows, "total", totalSoFar+rowsUpdated,
			"Workspace backfill batch")

		if tasks.WorkspaceBackfillBatchSleepMs > 0 {
			time.Sleep(time.Duration(tasks.WorkspaceBackfillBatchSleepMs) * time.Millisecond)
		}
	}

	return rowsUpdated, true, nil
}

func logPendingStats() error {
	pending, err := countPending()
	if err != nil {
		return err
	}
	invalid, err := countInvalidPending()
	if err != nil {
		return err
	}
	utils.LogInfo("pending", pending, "invalidPending", invalid, "Workspace backfill pending rows")
	if invalid > 0 {
		utils.LogWarn("invalidPending", invalid, "Rows with workspace_id NULL and invalid workspaces are skipped")
	}
	return nil
}

func loadPendingAccounts() ([]int, error) {
	var accounts []int
	err := tasks.WithReadReplicaTx(func(tx *gorm.DB) error {
		return tx.Raw(pendingAccountsSQL).Scan(&accounts).Error
	})
	return accounts, err
}

func countPending() (int64, error) {
	var cnt int64
	err := tasks.WithReadReplicaTx(func(tx *gorm.DB) error {
		return tx.Table("system_inventory").Where(pendingRowsSQL).Count(&cnt).Error
	})
	return cnt, err
}

func countInvalidPending() (int64, error) {
	var cnt int64
	err := tasks.WithReadReplicaTx(func(tx *gorm.DB) error {
		return tx.Table("system_inventory").Where(invalidPendingRowsSQL).Count(&cnt).Error
	})
	return cnt, err
}

func backfillBatch(rhAccountID, batchLimit int) (int64, error) {
	var rows int64
	err := tasks.CancelableDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(setReplicaRoleSQL).Error; err != nil {
			return err
		}
		res := tx.Exec(backfillUpdateSQL, rhAccountID, batchLimit)
		if res.Error != nil {
			return res.Error
		}
		rows = res.RowsAffected
		return nil
	})
	return rows, err
}
