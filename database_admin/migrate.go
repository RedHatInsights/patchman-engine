package database_admin //nolint:revive,stylecheck

import (
	"app/base/utils"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // we load migrations from local folder
	"github.com/lib/pq"
)

func NewConn(databaseURL string) (database.Driver, *sql.DB, error) {
	baseConn, err := pq.NewConnector(databaseURL)
	if err != nil {
		return nil, nil, err
	}

	loggingConn := pq.ConnectorWithNoticeHandler(baseConn, func(e *pq.Error) {
		fmt.Printf("Notice: %v\n", e)
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
	forceMigrationVersion := utils.Getenv("FORCE_SCHEMA_VERSION", "")
	schemaMigration := utils.GetIntEnvOrDefault("SCHEMA_MIGRATION", -1)
	if forceMigrationVersion != "" {
		// reset dirty flag and force set the current schema version
		var ver int
		ver, err = strconv.Atoi(forceMigrationVersion)
		if err == nil {
			err = m.Force(ver)
		}
	}

	if err == nil {
		if schemaMigration < 0 {
			err = m.Up()
		} else {
			err = m.Migrate(uint(schemaMigration))
		}
	}

	if err != nil && err.Error() == "no change" {
		fmt.Println("no change")
		return
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error upgrading the database: %v", err.Error())
		panic(err)
	}
}

func isUpgraded(conn database.Driver, sourceURL string) bool {
	m := createMigrate(conn, sourceURL)
	expectedSchema := utils.GetIntEnvOrDefault("SCHEMA_MIGRATION", -1)
	fmt.Printf("DB migration in progress, waiting for schema=%d\n", expectedSchema)

	curVersion, dirty, err := m.Version()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current DB version: %v\n", err.Error())
		return false
	}
	if dirty {
		fmt.Fprintln(os.Stderr, "Previous migration failed - dirty=true")
		return false
	}
	if int(curVersion) == expectedSchema {
		fmt.Println("DB is upgraded")
		return true
	}

	maxVer := 0
	dir := sourceURL[len("file://"):]
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading migration files %s in %s: %v\n", files, dir, err.Error())
		return false
	}
	for _, f := range files {
		ver, _, _ := cut(f.Name(), '_')
		intVer, err := strconv.Atoi(ver)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't convert %s to integer: %v\n", ver, err)
			return false
		}
		if intVer > maxVer {
			maxVer = intVer
		}
	}

	fmt.Printf("current version: %d, expected: %d\n", curVersion, maxVer)
	return int(curVersion) == maxVer
}

type logger struct{}

func (t logger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (t logger) Verbose() bool {
	return true
}

func createMigrate(conn database.Driver, sourceURL string) *migrate.Migrate {
	m, err := migrate.NewWithDatabaseInstance(sourceURL, utils.FailIfEmpty(utils.Cfg.DBName, "DB_NAME"), conn)
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
