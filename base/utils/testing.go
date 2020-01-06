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

func SkipWithoutPlatform(t *testing.T) {
	if os.Getenv("VMAAS_ADDRESS") == "" {
		t.Skip("testing platform instance not used - skipping")
	}
}
