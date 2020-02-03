package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/utils"
	"testing"
)

func TestMarkSystemsStale(t *testing.T) {
	utils.SkipWithoutDB(t)
}
