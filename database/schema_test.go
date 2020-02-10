package database

import (
	"app/base/database"
	"app/base/utils"
	"fmt"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func setCmdAuth(cmd *exec.Cmd) {
	cmd.Args = append(cmd.Args,
		"-h", utils.GetenvOrFail("DB_HOST"),
		"-p", utils.GetenvOrFail("DB_PORT"),
		"-U", utils.GetenvOrFail("DB_USER"),
		"-d", utils.GetenvOrFail("DB_NAME"))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%v", utils.GetenvOrFail("DB_PASSWD")))
}

func TestSchemaCompatiblity(t *testing.T) {
	utils.SkipWithoutDB(t)
	cfg := postgres.Config{
		DatabaseName:    "patchman",
		SchemaName:      "public",
		MigrationsTable: "schema_migrations",
	}
	database.Configure()

	dropAll := exec.Command("/usr/bin/psql", "-f", "./schema/clear_db.sql")
	setCmdAuth(dropAll)
	_, err := dropAll.CombinedOutput()
	assert.NoError(t, err)

	driver, err := postgres.WithInstance(database.Db.DB(), &cfg)
	assert.NoError(t, err)

	// Tests are run from local directory
	m, err := migrate.NewWithDatabaseInstance("file://migrations",
		utils.GetenvOrFail("DB_NAME"), driver)
	assert.Nil(t, err)

	err = m.Up()
	assert.NoError(t, err)

	dumpCmd := exec.Command("pg_dump", "-O")
	setCmdAuth(dumpCmd)

	migrated, err := dumpCmd.Output()
	assert.NoError(t, err)
	err = m.Drop()
	assert.NoError(t, err)

	rawCreate := exec.Command("/usr/bin/psql", "-f", "./schema/create_schema.sql")
	setCmdAuth(rawCreate)

	_, err = rawCreate.CombinedOutput()
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
