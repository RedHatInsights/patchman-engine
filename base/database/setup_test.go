package database

import (
	"os"
	"testing"
)

func SkipMissingPostgres(t *testing.T) {
	if os.Getenv("DB_TYPE") != "postgres" {
		t.Skip(" Non-PostgreSQL config - skipping")
	}
}

func TestDBCheck(t *testing.T) {
	SkipMissingPostgres(t)

	config := loadEnvPostgreSQLConfig("DB_")
	conn := openPostgreSQL(config)

	check(conn)
}

func TestTestingConfig(t *testing.T) {
	ConfigureSQLite()
}
