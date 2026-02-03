package database_admin

import (
	"app/base/utils"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // we load migrations from local folder
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

const (
	BLOCK    = 1
	CONTINUE = 2
	MIGRATE  = 3
)

func NewConn(databaseURL string) (database.Driver, *sql.DB, error) {
	baseConn, err := pq.NewConnector(databaseURL)
	if err != nil {
		return nil, nil, err
	}

	loggingConn := pq.ConnectorWithNoticeHandler(baseConn, func(e *pq.Error) {
		utils.LogInfo("Notice: " + e.Error())
	})

	db := sql.OpenDB(loggingConn)
	if _, err = db.Exec("SET client_min_messages TO NOTICE"); err != nil {
		return nil, nil, err
	}
	var driver database.Driver
	if driver, err = postgres.WithInstance(db, &postgres.Config{}); err != nil {
		return nil, nil, err
	}
	return driver, db, nil
}

func MigrateUp(conn database.Driver, sourceURL string) {
	var err error
	m := createMigrate(conn, sourceURL)
	if forceMigrationVersion > 0 {
		// reset dirty flag and force set the current schema version
		err = m.Force(forceMigrationVersion)
	}

	if err == nil {
		if schemaMigration < 0 {
			err = m.Up()
		} else {
			err = m.Migrate(uint(schemaMigration))
		}
	}

	if err != nil && err.Error() == "no change" {
		utils.LogInfo("no change")
		return
	}

	if err != nil {
		utils.LogError("err", err.Error(), "error upgrading the database")
		panic(err)
	}
}

func latestSchemaMigrationFileVersion(sourceURL string) (int, error) {
	latestVer := 0
	dir := sourceURL[len("file://"):]
	files, err := os.ReadDir(filepath.Clean(dir))
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("Error reading migration files %s in %s", files, dir))
	}
	for _, f := range files {
		ver, _, _ := cut(f.Name(), '_')
		intVer, err := strconv.Atoi(ver)
		if err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("Can't convert %s to integer: %v\n", ver, err))
		}
		if intVer > latestVer {
			latestVer = intVer
		}
	}
	return latestVer, nil
}

func dbSchemaVersion(conn database.Driver, sourceURL string) (int, error) {
	m := createMigrate(conn, sourceURL)
	curVersion, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		// no schema yet
		return 0, nil
	}
	if err != nil {
		return 0, errors.Wrap(err, "Error getting current DB version")
	}
	if dirty {
		return 0, migrate.ErrDirty{Version: int(curVersion)} //nolint:gosec
	}
	return int(curVersion), nil //nolint:gosec
}

func migrateAction(conn database.Driver, sourceURL string) int {
	expectedSchema := schemaMigration
	utils.LogInfo("schema", expectedSchema, "DB migration in progress, waiting for schema")
	dbSchema, err := dbSchemaVersion(conn, sourceURL)
	if err != nil {
		if errors.As(err, &migrate.ErrDirty{}) && forceMigrationVersion > 0 {
			return MIGRATE
		}
		utils.LogError("err", err.Error(), "error getting current DB version")
		return BLOCK
	}
	migrationSchema, err := latestSchemaMigrationFileVersion(sourceURL)
	if err != nil {
		utils.LogError("err", err.Error(), "failed to load latestSchemaMigrationFileVersion")
		return BLOCK
	}

	if migrationSchema < expectedSchema || migrationSchema < dbSchema {
		// some migration files are missing
		utils.LogError("schema", migrationSchema, "missing migration files for schema")
		return BLOCK
	}
	if dbSchema == expectedSchema && migrationSchema > dbSchema {
		// schema can be upgraded but is intentionaly blocked by SCHEMA_MIGRATION
		utils.LogWarn("deployment blocked, enable migrations to proceed")
		return BLOCK
	}
	if dbSchema == migrationSchema &&
		(dbSchema == expectedSchema || expectedSchema == -1) {
		utils.LogInfo("DB is upgraded")
		return CONTINUE
	}
	utils.LogInfo("current version:", dbSchema, "expected: ", expectedSchema, "migrateAction")
	return MIGRATE
}

type logger struct{}

func (t logger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (t logger) Verbose() bool {
	return true
}

func createMigrate(conn database.Driver, sourceURL string) *migrate.Migrate {
	m, err := migrate.NewWithDatabaseInstance(sourceURL, utils.FailIfEmpty(utils.CoreCfg.DBName, "DB_NAME"), conn)
	if err != nil {
		panic(err)
	}

	m.Log = logger{}
	return m
}

func cut(s string, what byte) (before, after string, found bool) {
	index := strings.IndexByte(s, what)
	if index < 0 {
		return "", s, false
	}
	return s[:index], s[index:], true
}
