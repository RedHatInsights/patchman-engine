package database_admin //nolint:revive,stylecheck

import (
	"app/base/database"
	"app/base/utils"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func setCmdAuth(cmd *exec.Cmd) {
	cmd.Args = append(cmd.Args,
		"-h", utils.FailIfEmpty(utils.Cfg.DBHost, "DB_HOST"),
		"-p", strconv.Itoa(utils.Cfg.DBPort),
		"-U", utils.FailIfEmpty(utils.Cfg.DBUser, "DB_USER"),
		"-d", utils.FailIfEmpty(utils.Cfg.DBName, "DB_NAME"))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%v", utils.FailIfEmpty(utils.Cfg.DBPassword, "DB_PASSWD")))
}

func TestSchemaCompatiblity(t *testing.T) {
	utils.SkipWithoutDB(t)
	cfg := postgres.Config{
		DatabaseName:    "patchman",
		SchemaName:      "public",
		MigrationsTable: "schema_migrations",
	}
	database.Configure()

	err := database.ExecFile("./schema/clear_db.sql")
	assert.NoError(t, err)

	sqlDB, _ := database.Db.DB()
	driver, err := postgres.WithInstance(sqlDB, &cfg)
	assert.NoError(t, err)

	// Tests are run from local directory
	m, err := migrate.NewWithDatabaseInstance("file://migrations",
		utils.FailIfEmpty(utils.Cfg.DBName, "DB_NAME"), driver)
	assert.Nil(t, err)

	err = m.Up()
	assert.NoError(t, err)

	dumpCmd := exec.Command("pg_dump", "-O")
	setCmdAuth(dumpCmd)

	migrated, err := dumpCmd.Output()
	assert.NoError(t, err)
	err = m.Drop()
	assert.NoError(t, err)

	err = database.ExecFile("./schema/create_schema.sql")

	assert.NoError(t, err)

	dumpCmd = exec.Command("pg_dump", "-O")
	setCmdAuth(dumpCmd)

	fromScratch, err := dumpCmd.Output()
	assert.NoError(t, err)

	migratedLines := strings.SplitAfter(string(migrated), "\n")
	scratchLines := strings.SplitAfter(string(fromScratch), "\n")

	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{A: migratedLines, B: scratchLines})
	assert.NoError(t, err)

	// nolint:gosec
	if len(diff) > 0 {
		fmt.Print(diff)
		if err := os.WriteFile("/tmp/schema-1-migrated.dump", migrated, 0600); err != nil {
			utils.LogError(err)
		}
		if err := os.WriteFile("/tmp/schema-2-fromscratch.dump", fromScratch, 0600); err != nil {
			utils.LogError(err)
		}
	}
	assert.Equal(t, len(diff), 0)
}

func TestSchemaEmptyText(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	var cols []string
	query := `SELECT c.relname || '.' || a.attname AS "col"
				FROM pg_catalog.pg_class c
				JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			   WHERE relkind in ( 'r', 'p')
				 AND relispartition = false
				 AND pg_catalog.pg_table_is_visible(c.oid)
				 AND n.nspname = 'public'
				 AND a.attnum > 0                           -- skip system columns
				 AND NOT a.attisdropped                     -- skip dropped columns
				 AND (a.atttypid = 1043 OR a.atttypid = 25) -- filter only varchars/text
					 -- skip cols that already has this constraint
				 AND NOT EXISTS ( SELECT 1
									FROM pg_catalog.pg_constraint
								   WHERE conname = c.relname || '_' || a.attname || '_check'
								);`
	err := database.Db.Raw(query).Find(&cols).Error
	assert.NoError(t, err)
	var msg string
	for _, col := range cols {
		msg += fmt.Sprintf("\nMissing empty() constraint on column '%s'", col)
	}
	assert.Empty(t, msg)
}

func TestMigrateAction(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()
	conn, _ := dbConn()
	sourceURL := "file://migrations"
	update := "UPDATE schema_migrations SET version = ?"
	origDBSchema, err := dbSchemaVersion(conn, sourceURL)
	assert.Nil(t, err)
	origMigrationSchema, err := latestSchemaMigrationFileVersion(sourceURL)
	assert.Nil(t, err)
	origSchemaMigration := os.Getenv("SCHEMA_MIGRATION")
	// db has higher version then migration files

	os.Setenv("SCHEMA_MIGRATION", fmt.Sprint(origMigrationSchema+100))
	what := migrateAction(conn, sourceURL)
	assert.Equal(t, BLOCK, what)

	// db is actual but there are new migrations
	os.Setenv("SCHEMA_MIGRATION", "1")
	assert.Nil(t, database.Db.Exec(update, 1).Error)
	what = migrateAction(conn, sourceURL)
	assert.Equal(t, BLOCK, what)

	// db is actual
	os.Setenv("SCHEMA_MIGRATION", "-1")
	assert.Nil(t, database.Db.Exec(update, origMigrationSchema).Error)
	what = migrateAction(conn, sourceURL)
	assert.Equal(t, CONTINUE, what)
	// db is actual
	os.Setenv("SCHEMA_MIGRATION", fmt.Sprint(origMigrationSchema))
	what = migrateAction(conn, sourceURL)
	assert.Equal(t, CONTINUE, what)

	// db is actual
	os.Setenv("SCHEMA_MIGRATION", fmt.Sprint(origMigrationSchema))
	assert.Nil(t, database.Db.Exec(update, origMigrationSchema-1).Error)
	what = migrateAction(conn, sourceURL)
	assert.Equal(t, MIGRATE, what)

	// cleanup
	os.Setenv("SCHEMA_MIGRATION", origSchemaMigration)
	assert.Nil(t, database.Db.Exec(update, origDBSchema).Error)
}
