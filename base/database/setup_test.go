package database

import (
	"os"
	"testing"
)

func TestConfigFromEnvsd(t *testing.T) {
	if os.Getenv("DB_TYPE") != "mysql" {
		t.Skip(" Non-MySQL config - skipping")
	}

	loadEnvPostgreSQLConfig()
}

func TestDBCheck(t *testing.T) {
	if os.Getenv("DB_TYPE") != "mysql" {
		t.Skip(" Non-MySQL config - skipping")
	}

	config := loadEnvPostgreSQLConfig()
	conn := openPostgreSQL(config)

	check(conn)
}

func TestTestingConfig(t *testing.T) {
	ConfigureSQLite()
}
