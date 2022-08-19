package database_admin //nolint:revive,stylecheck

import (
	"app/base/utils"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4/database"
	log "github.com/sirupsen/logrus"
)

func execOrPanic(db *sql.DB, query string, args ...interface{}) {
	if _, err := db.Exec(query, args...); err != nil {
		panic(err)
	}
}

func execFromFile(db *sql.DB, filepath string) {
	query, err := ioutil.ReadFile(filepath)
	if err != nil {
		panic(err)
	}
	execOrPanic(db, string(query))
}

func getAdvisoryLock(db *sql.DB) {
	log.Info("Getting advisory lock")
	execOrPanic(db, "SELECT pg_advisory_lock(123)")
}

func releaseAdvisoryLock(db *sql.DB) {
	log.Info("Releasing advisory lock")
	execOrPanic(db, "SELECT pg_advisory_unlock(123)")
}

// Wait for closing of all "listener", "evaluator" and "vmaas_sync" database sessions.
func waitForSessionClosed(db *sql.DB) {
	for {
		session := ""
		err := db.QueryRow(
			"SELECT usename || ' ' || substring(query for 50) FROM pg_stat_activity WHERE " +
				"usename IN ('evaluator', 'listener', 'vmaas_sync') LIMIT 30;",
		).Scan(&session)
		if err != nil {
			log.Info(err)
		}
		if session == "" {
			log.Info("No 'listener', 'evaluator', 'vmaas_sync' sessions found")
			return
		}
		utils.Log("session:", session).Info("Session found")
		time.Sleep(time.Second)
	}
}

func setPgEnv() {
	os.Setenv("PGHOST", utils.Cfg.DBHost)
	os.Setenv("PGUSER", utils.Cfg.DBAdminUser)
	os.Setenv("PGPASSWORD", utils.Cfg.DBAdminPassword)
	os.Setenv("PGDATABASE", utils.Cfg.DBName)
	os.Setenv("PGPORT", fmt.Sprint(utils.Cfg.DBPort))
	os.Setenv("PGSSLMODE", utils.Cfg.DBSslMode)
	os.Setenv("PGSSLROOTCERT", utils.Cfg.DBSslRootCert)
}

func startMigration(conn database.Driver, db *sql.DB, migrationFilesURL string) {
	log.Info("Blocking writing users during the migration")
	execOrPanic(db, "ALTER USER listener NOLOGIN")
	execOrPanic(db, "ALTER USER evaluator NOLOGIN")
	execOrPanic(db, "ALTER USER vmaas_sync NOLOGIN")
	waitForSessionClosed(db)

	MigrateUp(conn, migrationFilesURL)

	log.Info("Reverting components privileges")
	execOrPanic(db, "ALTER USER listener LOGIN")
	execOrPanic(db, "ALTER USER evaluator LOGIN")
	execOrPanic(db, "ALTER USER vmaas_sync LOGIN")
}

func dbConn() (database.Driver, *sql.DB) {
	sslModeCert := utils.Cfg.DBSslMode
	if utils.Cfg.DBSslRootCert != "" {
		sslModeCert += "&sslrootcert=" + utils.Cfg.DBSslRootCert
	}
	databaseURL := fmt.Sprintf("postgres://%s/%s?sslmode=%s", utils.Cfg.DBHost, utils.Cfg.DBName, sslModeCert)
	setPgEnv()

	conn, db, err := NewConn(databaseURL)
	if err != nil {
		panic(err)
	}
	return conn, db
}

func UpdateDB(migrationFilesURL string) {
	utils.ConfigureLogging()
	migrationEnabled := utils.GetBoolEnvOrDefault("ENABLE_MIGRATION", false)

	conn, db := dbConn()

	if !migrationEnabled {
		if isUpgraded(conn, migrationFilesURL) {
			return
		}
		utils.Log("ENABLE_MIGRATION", migrationEnabled).Info("Deployment blocked, enable migrations to proceed")
		// sleep until next deployment
		select {}
	}

	getAdvisoryLock(db)
	defer releaseAdvisoryLock(db)

	if utils.GetBoolEnvOrDefault("RESET_SCHEMA", false) {
		execOrPanic(db, "DROP SCHEMA IF EXISTS public CASCADE")
		execOrPanic(db, "CREATE SCHEMA IF NOT EXISTS public")
		execOrPanic(db, "GRANT ALL ON SCHEMA public TO ?", utils.Cfg.DBUser)
		execOrPanic(db, "GRANT ALL ON SCHEMA public TO public")
	}

	if utils.GetBoolEnvOrDefault("UPDATE_USERS", false) {
		log.Info("Creating application components users")
		execFromFile(db, "./database_admin/schema/create_users.sql")
	}

	if isUpgraded(conn, migrationFilesURL) {
		log.Info("Skipping migration")
	} else {
		log.Info("Migrating the database")
		startMigration(conn, db, migrationFilesURL)
	}

	if utils.GetBoolEnvOrDefault("UPDATE_USERS", false) {
		log.Info("Setting user passwords")
		// Set specific password for each user. If the users are already created, change the password.
		// This is performed on each startup in order to ensure users have latest pasword
		execOrPanic(db, "ALTER USER listener WITH PASSWORD '"+utils.GetenvOrFail("LISTENER_PASSWORD")+"'")
		execOrPanic(db, "ALTER USER evaluator WITH PASSWORD '"+utils.GetenvOrFail("EVALUATOR_PASSWORD")+"'")
		execOrPanic(db, "ALTER USER manager WITH PASSWORD '"+utils.GetenvOrFail("MANAGER_PASSWORD")+"'")
		execOrPanic(db, "ALTER USER vmaas_sync WITH PASSWORD '"+utils.GetenvOrFail("VMAAS_SYNC_PASSWORD")+"'")
		if utils.GetBoolEnvOrDefault("UPDATE_CYNDI_PASSWD", false) {
			execOrPanic(db, "ALTER USER cyndi WITH PASSWORD '"+utils.GetenvOrFail("CYNDI_PASSWORD")+"'")
		}
	}

	if utils.GetBoolEnvOrDefault("UPDATE_DB_CONFIG", false) {
		log.Info("Setting database config")
		execFromFile(db, "./database_admin/config.sql")
	}
}

func CheckUpgraded(sourceURL string) {
	conn, _ := dbConn()
	for {
		if isUpgraded(conn, sourceURL) {
			return
		}
		time.Sleep(time.Second)
	}
}
