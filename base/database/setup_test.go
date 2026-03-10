package database

import (
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDBCheck(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()
}

func TestAdditionalParams(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	assert.True(t, len(OtherAdvisoryTypes) == 2)
}
