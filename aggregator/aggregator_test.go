package aggregator

import (
	"app/base/database"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	exitCode := m.Run()
	if database.DB != nil {
		database.DB.Exec("SELECT refresh_account_advisory_caches_multi(NULL, 1)")
	}
	os.Exit(exitCode)
}
