package sql_job

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunWithoutScript(t *testing.T) {
	for _, value := range []string{"", " \t\n"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("SQL_JOB_FILE", value)
			require.NoError(t, run())
		})
	}
}

func TestRunInvalidScript(t *testing.T) {
	t.Setenv("SQL_JOB_FILE", "missing.sql")
	require.Error(t, run())
}

func TestReadScript(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "valid.sql"), []byte("SELECT 1; SELECT 2;"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "empty.sql"), []byte(" \n"), 0600))
	require.NoError(t, os.Symlink(filepath.Join(dir, "valid.sql"), filepath.Join(dir, "link.sql")))
	for _, name := range []string{
		"", "../valid.sql", "/valid.sql", `..\valid.sql`, "missing.sql", "empty.sql", "link.sql", "valid.txt",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := readScript(dir, name)
			require.Error(t, err)
		})
	}
	script, err := readScript(dir, "valid.sql")
	require.NoError(t, err)
	assert.Equal(t, "SELECT 1; SELECT 2;", string(script))
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	db, err := database.DB.DB()
	require.NoError(t, err)
	return db
}

func TestExecute(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "CREATE TABLE sql_job_test (value integer)")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := db.ExecContext(ctx, "DROP TABLE sql_job_test")
		require.NoError(t, err)
	})
	count := func(want int) {
		t.Helper()
		var got int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM sql_job_test").Scan(&got))
		assert.Equal(t, want, got)
	}
	run := func(script string) error {
		return execute(ctx, db, []byte(script), time.Second, time.Second)
	}
	require.NoError(t, run("INSERT INTO sql_job_test VALUES (1); INSERT INTO sql_job_test VALUES (2);"))
	count(2)
	require.Error(t, run("INSERT INTO sql_job_test VALUES (3); SELECT 1 / 0;"))
	count(2)
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck
	_, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", advisoryLockID)
	require.NoError(t, err)
	require.ErrorContains(t, run("INSERT INTO sql_job_test VALUES (4);"), "another SQL job")
	require.NoError(t, tx.Rollback())
	count(2)
	require.Error(t, execute(ctx, db, []byte("INSERT INTO sql_job_test VALUES (5); SELECT pg_sleep(1);"),
		time.Second, 10*time.Millisecond))
	count(2)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	require.Error(t, execute(canceled, db, []byte("INSERT INTO sql_job_test VALUES (6);"), time.Second, time.Second))
	count(2)
	deadline, cancelDeadline := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancelDeadline()
	require.Error(t, execute(deadline, db, []byte("INSERT INTO sql_job_test VALUES (7); SELECT pg_sleep(1);"),
		time.Second, time.Second))
	count(2)
	require.Error(t, execute(ctx, db, []byte("SELECT 1;"), 0, time.Second))
}

func TestSmokeScript(t *testing.T) {
	script, err := readScript("sql", "000_test.sql")
	require.NoError(t, err)
	db := testDB(t)
	require.NoError(t, execute(context.Background(), db, script, time.Second, time.Second))
}
