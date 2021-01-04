package database_admin //nolint:golint,stylecheck

import (
	"app/base/utils"
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // we load migrations from local folder
	"github.com/lib/pq"
	"os"
)

func NewConn(databaseURL string) (database.Driver, error) {
	baseConn, err := pq.NewConnector(databaseURL)
	if err != nil {
		return nil, err
	}

	loggingConn := pq.ConnectorWithNoticeHandler(baseConn, func(e *pq.Error) {
		fmt.Printf("Notice: %v\n", e)
	})

	db := sql.OpenDB(loggingConn)
	if _, err = db.Exec("SET client_min_messages TO NOTICE"); err != nil {
		return nil, err
	}
	var driver database.Driver
	if driver, err = postgres.WithInstance(db, &postgres.Config{}); err != nil {
		return nil, err
	}
	return driver, nil
}

func MigrateUp(sourceURL, databaseURL string) {
	conn, err := NewConn(databaseURL)
	if err != nil {
		panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance(sourceURL, utils.GetenvOrFail("DB_NAME"), conn)
	if err != nil {
		panic(err)
	}

	m.Log = logger{}

	schemaMigration := utils.GetIntEnvOrDefault("SCHEMA_MIGRATION", -1)
	if schemaMigration < 0 {
		err = m.Up()
	} else {
		err = m.Migrate(uint(schemaMigration))
	}

	if err != nil && err.Error() == "no change" {
		fmt.Println("no change")
		return
	}

	if err != nil {
		// Don't panic on error, Log and keep the container running so we can diagnose it
		fmt.Fprintf(os.Stderr, "Error upgrading the database: %v", err.Error())
	}
}

type logger struct{}

func (t logger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (t logger) Verbose() bool {
	return true
}
