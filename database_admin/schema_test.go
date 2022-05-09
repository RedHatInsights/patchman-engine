package database_admin //nolint:revive,stylecheck

import (
	"app/base/database"
	"app/base/utils"
	"fmt"
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

	fmt.Print(diff)
	assert.Equal(t, len(diff), 0)
}
