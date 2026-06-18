package database_admin

import (
	"app/base/utils"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var lockUsers = []string{"listener", "evaluator", "manager", "vmaas_sync"}

const activeAppSessionsQuery = `SELECT usename || ' ' || substring(query for 50)
FROM pg_stat_activity
WHERE usename = ANY($1)
LIMIT 1`

const sessionCheckMaxRetries = 5

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

// findActiveAppSession returns the first open session for lockUsers, if any.
func findActiveAppSession(db *sql.DB) (session string, found bool, err error) {
	err = db.QueryRow(activeAppSessionsQuery, pq.Array(lockUsers)).Scan(&session)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return session, true, nil
}

// Wait for closing of all lockUsers database sessions.
func waitForSessionClosed(db *sql.DB) {
	errRetries := 0
	for {
		session, found, err := findActiveAppSession(db)
		if err != nil {
			errRetries++
			utils.LogError("err", err.Error(), "attempt", errRetries, "failed to check app database sessions")
			if errRetries >= sessionCheckMaxRetries {
				panic(fmt.Errorf("failed to check app database sessions after %d attempts: %w",
					sessionCheckMaxRetries, err))
			}
			time.Sleep(time.Second)
			continue
		}
		errRetries = 0
		if !found {
			log.Info("No ", strings.Join(lockUsers, ", "), " sessions found")
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

func blockUsers(db *sql.DB) {
	for _, user := range lockUsers {
		execOrPanic(db, "ALTER USER "+user+" NOLOGIN")
	}
}

func unblockUsers(db *sql.DB) {
	for _, user := range lockUsers {
		execOrPanic(db, "ALTER USER "+user+" LOGIN")
	}
}

func startMigration(conn database.Driver, db *sql.DB, migrationFilesURL string) {
	log.Info("Blocking writing users during the migration")
	blockUsers(db)
	waitForSessionClosed(db)

	MigrateUp(conn, migrationFilesURL)

	log.Info("Reverting components privileges")
	unblockUsers(db)
}

func dbConn() (database.Driver, *sql.DB) {
	sslModeCert := utils.CoreCfg.DBSslMode
	if utils.CoreCfg.DBSslRootCert != "" {
		sslModeCert += "&sslrootcert=" + utils.CoreCfg.DBSslRootCert
	}
	databaseURL := fmt.Sprintf("postgres://%s/%s?sslmode=%s", utils.CoreCfg.DBHost, utils.CoreCfg.DBName, sslModeCert)
	databaseURL += "&tcp_keepalives_idle=60&tcp_keepalives_interval=30&tcp_keepalives_count=5"
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

	if unlockUsers {
		log.Info("Unlocking application components users")
		unblockUsers(db)
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
