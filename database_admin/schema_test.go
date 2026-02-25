package database_admin

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

func dumpSchema() ([]byte, error) {
	cmd := exec.Command("pg_dump", "--restrict-key", "testupgradekey", "-O")
	cmd.Args = append(cmd.Args,
		"-h", utils.FailIfEmpty(utils.CoreCfg.DBHost, "DB_HOST"),
		"-p", strconv.Itoa(utils.CoreCfg.DBPort),
		"-U", utils.FailIfEmpty(utils.CoreCfg.DBUser, "DB_USER"),
		"-d", utils.FailIfEmpty(utils.CoreCfg.DBName, "DB_NAME"))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%v", utils.FailIfEmpty(utils.CoreCfg.DBPassword, "DB_PASSWD")))
	return cmd.Output()
}

func writeTemp(dir, filename string, data []byte) {
	file, err := os.CreateTemp(dir, filename)
	if err != nil {
		utils.LogError(err)
		return
	}
	if _, err := file.Write(data); err != nil {
		utils.LogError(err)
	}
	file.Close()
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

	sqlDB, _ := database.DB.DB()
	driver, err := postgres.WithInstance(sqlDB, &cfg)
	assert.NoError(t, err)

	// Tests are run from local directory
	m, err := migrate.NewWithDatabaseInstance("file://migrations",
		utils.FailIfEmpty(utils.CoreCfg.DBName, "DB_NAME"), driver)
	assert.Nil(t, err)

	err = m.Up()
	assert.NoError(t, err)

	migrated, err := dumpSchema()
	assert.NoError(t, err)
	err = m.Drop()
	assert.NoError(t, err)

	err = database.ExecFile("./schema/create_schema.sql")

	assert.NoError(t, err)
	fromScratch, err := dumpSchema()
	assert.NoError(t, err)

	migratedLines := strings.SplitAfter(string(migrated), "\n")
	scratchLines := strings.SplitAfter(string(fromScratch), "\n")

	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{A: migratedLines, B: scratchLines})
	assert.NoError(t, err)

	if len(diff) > 0 {
		fmt.Print(diff)
		writeTemp("/tmp", "schema-1-migrated.*.dump", migrated)
		writeTemp("/tmp", "schema-2-fromscratch.*.dump", fromScratch)
	}
	assert.Equal(t, 0, len(diff))
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
	err := database.DB.Raw(query).Find(&cols).Error
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
	origSchemaMigration := schemaMigration
	// db has higher version then migration files

	schemaMigration = origMigrationSchema + 100
	what := migrateAction(conn, sourceURL)
	assert.Equal(t, BLOCK, what)

	// db is actual but there are new migrations
	schemaMigration = 1
	assert.Nil(t, database.DB.Exec(update, 1).Error)
	what = migrateAction(conn, sourceURL)
	assert.Equal(t, BLOCK, what)

	// db is actual
	schemaMigration = -1
	assert.Nil(t, database.DB.Exec(update, origMigrationSchema).Error)
	what = migrateAction(conn, sourceURL)
	assert.Equal(t, CONTINUE, what)
	// db is actual
	schemaMigration = origMigrationSchema
	what = migrateAction(conn, sourceURL)
	assert.Equal(t, CONTINUE, what)

	// db is actual
	schemaMigration = origMigrationSchema
	assert.Nil(t, database.DB.Exec(update, origMigrationSchema-1).Error)
	what = migrateAction(conn, sourceURL)
	assert.Equal(t, MIGRATE, what)

	// cleanup
	schemaMigration = origSchemaMigration
	assert.Nil(t, database.DB.Exec(update, origDBSchema).Error)
}
