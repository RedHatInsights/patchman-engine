package database_admin //nolint:revive,stylecheck

import (
	"app/base/utils"
	"database/sql"
	"fmt"
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
	query, err := os.ReadFile(filepath)
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
		utils.LogInfo("session:", session, "Session found")
		time.Sleep(time.Second)
	}
}

func setPgEnv() {
	os.Setenv("PGHOST", utils.CoreCfg.DBHost)
	os.Setenv("PGUSER", utils.CoreCfg.DBAdminUser)
	os.Setenv("PGPASSWORD", utils.CoreCfg.DBAdminPassword)
	os.Setenv("PGDATABASE", utils.CoreCfg.DBName)
	os.Setenv("PGPORT", fmt.Sprint(utils.CoreCfg.DBPort))
	os.Setenv("PGSSLMODE", utils.CoreCfg.DBSslMode)
	os.Setenv("PGSSLROOTCERT", utils.CoreCfg.DBSslRootCert)
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
	sslModeCert := utils.CoreCfg.DBSslMode
	if utils.CoreCfg.DBSslRootCert != "" {
		sslModeCert += "&sslrootcert=" + utils.CoreCfg.DBSslRootCert
	}
	databaseURL := fmt.Sprintf("postgres://%s/%s?sslmode=%s", utils.CoreCfg.DBHost, utils.CoreCfg.DBName, sslModeCert)
	setPgEnv()

	conn, db, err := NewConn(databaseURL)
	if err != nil {
		panic(err)
	}
	return conn, db
}

func UpdateDB(migrationFilesURL string) {
	utils.ConfigureLogging()
	conn, db := dbConn()

	getAdvisoryLock(db)
	defer releaseAdvisoryLock(db)

	if resetSchema {
		execOrPanic(db, "DROP SCHEMA IF EXISTS public CASCADE")
		execOrPanic(db, "CREATE SCHEMA IF NOT EXISTS public")
		execOrPanic(db, "GRANT ALL ON SCHEMA public TO ?", utils.CoreCfg.DBUser)
		execOrPanic(db, "GRANT ALL ON SCHEMA public TO public")
	}

	if updateUsers {
		log.Info("Creating application components users")
		execFromFile(db, "./database_admin/schema/create_users.sql")
	}

	switch action := migrateAction(conn, migrationFilesURL); action {
	case BLOCK:
		// sleep until next deployment
		releaseAdvisoryLock(db)
		select {}
	case CONTINUE:
		log.Info("Skipping migration")
	case MIGRATE:
		log.Info("Migrating the database")
		startMigration(conn, db, migrationFilesURL)
	}

	if updateUsers {
		log.Info("Setting user passwords")
		// Set specific password for each user. If the users are already created, change the password.
		// This is performed on each startup in order to ensure users have latest pasword
		execOrPanic(db, "ALTER USER listener WITH PASSWORD '"+utils.GetenvOrFail("LISTENER_PASSWORD")+"'")
		execOrPanic(db, "ALTER USER evaluator WITH PASSWORD '"+utils.GetenvOrFail("EVALUATOR_PASSWORD")+"'")
		execOrPanic(db, "ALTER USER manager WITH PASSWORD '"+utils.GetenvOrFail("MANAGER_PASSWORD")+"'")
		execOrPanic(db, "ALTER USER vmaas_sync WITH PASSWORD '"+utils.GetenvOrFail("VMAAS_SYNC_PASSWORD")+"'")
		if updateCyndiPasswd {
			execOrPanic(db, "ALTER USER cyndi WITH PASSWORD '"+utils.GetenvOrFail("CYNDI_PASSWORD")+"'")
		}
	}

	if updateDBConfig {
		log.Info("Setting database config")
		execFromFile(db, "./database_admin/config.sql")
	}
}

func CheckUpgraded(sourceURL string) {
	conn, _ := dbConn()
	for i := 0; i < 60; i++ {
		action := migrateAction(conn, sourceURL)
		if action == CONTINUE {
			return
		}
		time.Sleep(5 * time.Second)
	}
	fmt.Fprintln(os.Stderr, "Upgrade check aborted")
	os.Exit(1)
}
