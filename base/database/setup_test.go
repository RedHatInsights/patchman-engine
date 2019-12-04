package database

import (
	"app/base/utils"
	"testing"
)

func TestDBCheck(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()
}
