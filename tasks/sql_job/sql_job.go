package sql_job

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	scriptDirectory = "tasks/sql_job/sql"
	// Separate from the schema migration advisory lock, 123.
	advisoryLockID = 74201901
)

func readScript(directory, name string) ([]byte, error) {
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\`) || filepath.Ext(name) != ".sql" {
		return nil, fmt.Errorf("SQL_JOB_FILE must name a bundled .sql file, not a path")
	}
	path := filepath.Join(directory, name)
	info, err := os.Lstat(path) // #nosec G703 -- name is a single filename; directory is repository-owned.
	if err != nil {
		return nil, fmt.Errorf("stat SQL file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("SQL file must be a regular file")
	}
	script, err := os.ReadFile(path) // #nosec G703 -- validated filename and regular file in a trusted directory.
	if err != nil {
		return nil, fmt.Errorf("read SQL file: %w", err)
	}
	if strings.TrimSpace(string(script)) == "" {
		return nil, fmt.Errorf("SQL file is empty")
	}
	return script, nil
}

// execute runs trusted, repository-owned SQL. Scripts must not contain transaction
// control or psql commands. The transaction owns the advisory lock until completion.
func execute(ctx context.Context, db *sql.DB, script []byte, lockTimeout, statementTimeout time.Duration) error {
	if lockTimeout <= 0 || statementTimeout <= 0 {
		return fmt.Errorf("SQL job timeouts must be positive")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	var locked bool
	if err = tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock($1)", advisoryLockID).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("another SQL job holds the advisory lock")
	}
	if _, err = tx.ExecContext(ctx, `SELECT set_config('lock_timeout', $1, true),
		set_config('statement_timeout', $2, true)`,
		fmt.Sprintf("%dms", lockTimeout.Milliseconds()), fmt.Sprintf("%dms", statementTimeout.Milliseconds())); err != nil {
		return err
	}
	// No parameters: PostgreSQL's simple query protocol supports complete SQL files.
	if _, err = tx.ExecContext(ctx, string(script)); err != nil {
		return err
	}
	return tx.Commit()
}

func run() error {
	name := strings.TrimSpace(os.Getenv("SQL_JOB_FILE"))
	if name == "" {
		utils.LogInfo("No SQL script selected, skipping SQL job")
		return nil
	}
	script, err := readScript(scriptDirectory, name)
	if err != nil {
		return err
	}
	core.ConfigureApp()
	lockTimeout := time.Duration(utils.PodConfig.GetInt("sql_lock_timeout_seconds", 30)) * time.Second
	statementTimeout := time.Duration(utils.PodConfig.GetInt("sql_statement_timeout_seconds", 1500)) * time.Second
	ctx, cancel := context.WithTimeout(base.Context, statementTimeout)
	defer cancel()
	db, err := database.DB.DB()
	if err != nil {
		return err
	}
	start := time.Now()
	utils.LogInfo("file", name, "Starting SQL job")
	if err = execute(ctx, db, script, lockTimeout, statementTimeout); err != nil {
		return err
	}
	utils.LogInfo("file", name, "seconds", time.Since(start).Seconds(), "SQL job completed")
	return nil
}

func Run() {
	utils.ConfigureLogging()
	if err := run(); err != nil {
		utils.LogFatal("file", os.Getenv("SQL_JOB_FILE"), "err", err, "SQL job failed")
	}
}
