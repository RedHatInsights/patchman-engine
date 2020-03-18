package database_admin //nolint:golint,stylecheck

import (
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres database is used
	_ "github.com/golang-migrate/migrate/v4/source/file"       // we load migrations from local folder
)

func MigrateUp(sourceURL, databaseURL string) {
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		panic(err)
	}

	m.Log = logger{}
	err = m.Up()
	if err != nil {
		panic(err)
	}
}

type logger struct{}

func (t logger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (t logger) Verbose() bool {
	return true
}
