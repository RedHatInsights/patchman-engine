package utils

import (
	"os"
	"testing"
)


func SkipWithoutDB(t *testing.T) {
	if os.Getenv("USE_TESTING_DB") != "on" {
		t.Skip("testing database not used - skipping")
	}
}