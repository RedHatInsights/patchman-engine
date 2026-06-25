package database_admin

import (
	"app/base/database"
	"app/base/utils"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openAppDB(t *testing.T, user, password string) *sql.DB {
	t.Helper()
	sslModeCert := utils.CoreCfg.DBSslMode
	if utils.CoreCfg.DBSslRootCert != "" {
		sslModeCert += "&sslrootcert=" + utils.CoreCfg.DBSslRootCert
	}
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, password,
		utils.CoreCfg.DBHost, utils.CoreCfg.DBPort,
		utils.CoreCfg.DBName, sslModeCert,
	)
	db, err := sql.Open("postgres", url)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// Query errors must not be treated as "no sessions". Use an unreachable host/port so
// QueryRow fails on connect; sql.Open itself does not connect.
func TestFindActiveAppSessionInvalidDB(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://127.0.0.1:1/nope?sslmode=disable&connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, found, err := findActiveAppSession(db)
	assert.Error(t, err)
	assert.False(t, found)
}

func TestFindActiveAppSessionNoRows(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	_, db := dbConn()
	t.Cleanup(func() { _ = db.Close() })

	_, found, err := findActiveAppSession(db)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestFindActiveAppSessionFound(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	_ = openAppDB(t, "manager", utils.Getenv("MANAGER_PASSWORD", "manager"))

	_, db := dbConn()
	t.Cleanup(func() { _ = db.Close() })

	session, found, err := findActiveAppSession(db)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Contains(t, session, "manager")
}

func TestWaitForSessionClosedQueryErrors(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://127.0.0.1:1/nope?sslmode=disable&connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	assert.Panics(t, func() { waitForSessionClosed(db) })
}

func TestListActiveAppSessionsFound(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	_ = openAppDB(t, "manager", utils.Getenv("MANAGER_PASSWORD", "manager"))

	_, db := dbConn()
	t.Cleanup(func() { _ = db.Close() })

	sessions, err := listActiveAppSessions(db)
	require.NoError(t, err)

	found := false
	for _, session := range sessions {
		if session.usename == "manager" {
			found = true
			assert.NotZero(t, session.pid)
			break
		}
	}
	assert.True(t, found)
}

func TestTerminateAppSessions(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	listenerDB := openAppDB(t, "listener", utils.Getenv("LISTENER_PASSWORD", "listener"))
	listenerDB.SetMaxOpenConns(1)

	_, db := dbConn()
	t.Cleanup(func() {
		unblockUsers(db)
		_ = db.Close()
	})

	var listenerPID int
	require.NoError(t, listenerDB.QueryRow("SELECT pg_backend_pid()").Scan(&listenerPID))

	sessions, err := listActiveAppSessions(db)
	require.NoError(t, err)
	found := false
	for _, session := range sessions {
		if session.pid == listenerPID {
			found = true
			break
		}
	}
	require.True(t, found)

	blockUsers(db)
	terminateAppSessions(db)

	var one int
	err = listenerDB.QueryRow("SELECT 1").Scan(&one)
	assert.Error(t, err)
}

func TestStartMigrationBeforeMigrateUp(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	oldTerminate := terminateDBSessions
	terminateDBSessions = true
	t.Cleanup(func() { terminateDBSessions = oldTerminate })

	listenerDB := openAppDB(t, "listener", utils.Getenv("LISTENER_PASSWORD", "listener"))
	listenerDB.SetMaxOpenConns(1)

	_, db := dbConn()
	t.Cleanup(func() {
		unblockUsers(db)
		_ = db.Close()
	})

	prepareForMigration(db)

	var one int
	err := listenerDB.QueryRow("SELECT 1").Scan(&one)
	assert.Error(t, err)

	_, found, err := findActiveAppSession(db)
	require.NoError(t, err)
	assert.False(t, found)

	for _, user := range lockUsers {
		var canLogin bool
		err := db.QueryRow("SELECT rolcanlogin FROM pg_roles WHERE rolname = $1", user).Scan(&canLogin)
		require.NoError(t, err)
		assert.False(t, canLogin, "user %s should remain blocked before MigrateUp", user)
	}
}
