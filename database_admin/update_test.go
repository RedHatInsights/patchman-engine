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
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, password,
		utils.CoreCfg.DBHost, utils.CoreCfg.DBPort,
		utils.CoreCfg.DBName, utils.CoreCfg.DBSslMode,
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
