package database

import (
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDBCheck(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()
}

func TestAdditionalParams(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	assert.True(t, len(AdvisoryTypes) == 5)
}
