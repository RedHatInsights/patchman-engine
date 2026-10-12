package sql_job

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBackfillOwnerID(t *testing.T) {
	db := testDB(t)
	script, err := readScript("sql", "001_backfill_owner_id.sql")
	require.NoError(t, err)
	// The temporary table shadows the real table on this transaction's connection.
	setup := `CREATE TEMP TABLE system_inventory (id integer, owner_id uuid, subscription_manager_id uuid) ON COMMIT DROP;
	INSERT INTO system_inventory VALUES
	(1, NULL, '11111111-1111-1111-1111-111111111111'),
	(2, '22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111'),
	(3, NULL, NULL);`
	verify := `DO $$ BEGIN
	IF (SELECT owner_id FROM system_inventory WHERE id = 1) IS DISTINCT FROM '11111111-1111-1111-1111-111111111111'::uuid
	OR (SELECT owner_id FROM system_inventory WHERE id = 2) IS DISTINCT FROM '22222222-2222-2222-2222-222222222222'::uuid
	OR (SELECT owner_id FROM system_inventory WHERE id = 3) IS NOT NULL
	OR EXISTS (SELECT 1 FROM system_inventory WHERE id IN (1, 2)
	AND subscription_manager_id IS DISTINCT FROM '11111111-1111-1111-1111-111111111111'::uuid)
	OR (SELECT subscription_manager_id FROM system_inventory WHERE id = 3) IS NOT NULL
	THEN RAISE EXCEPTION 'backfill changed unexpected values'; END IF;
	END $$;`
	require.NoError(t, execute(context.Background(), db, []byte(setup+string(script)+verify+string(script)+verify),
		time.Second, time.Second))
}

func TestOwnerIDMigration(t *testing.T) {
	db := testDB(t)
	dir := filepath.Join("..", "..", "database_admin", "migrations")
	up, err := readScript(dir, "169_add_owner_id.up.sql")
	require.NoError(t, err)
	down, err := readScript(dir, "169_add_owner_id.down.sql")
	require.NoError(t, err)
	setup := `CREATE TEMP TABLE system_inventory (id integer, subscription_manager_id uuid)
	PARTITION BY HASH (id);
	CREATE TEMP TABLE sql_job_migration_partition PARTITION OF system_inventory
	FOR VALUES WITH (MODULUS 1, REMAINDER 0);
	INSERT INTO system_inventory VALUES (1, '11111111-1111-1111-1111-111111111111');`
	verify := `DO $$ BEGIN
	IF (SELECT subscription_manager_id FROM system_inventory WHERE id = 1)
	IS DISTINCT FROM '11111111-1111-1111-1111-111111111111'::uuid
	OR (SELECT owner_id FROM sql_job_migration_partition WHERE id = 1) IS NOT NULL
	THEN RAISE EXCEPTION 'migration changed existing data'; END IF;
	END $$;`
	cleanup := `SELECT subscription_manager_id FROM system_inventory;
	DROP TABLE system_inventory;`
	require.NoError(t, execute(context.Background(), db, []byte(setup+string(up)+verify+string(down)+cleanup),
		time.Second, time.Second))
}
