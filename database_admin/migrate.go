package database_admin //nolint:golint,stylecheck

import (
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres database is used
	_ "github.com/golang-migrate/migrate/v4/source/file"       // we load migrations from local folder
	"os"
)

func MigrateUp(sourceURL, databaseURL string) {
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		panic(err)
	}

	m.Log = logger{}

	ver, dirty, err := m.Version()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error upgrading the database: %v", err)
	}

	if ver == 41 && dirty {
		if err := m.Force(40); err != nil {
			fmt.Fprintf(os.Stderr, "Error upgrading the database: %v", err)
		}
	}

	err = m.Up()
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
