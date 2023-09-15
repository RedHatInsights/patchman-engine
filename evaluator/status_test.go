package evaluator

import (
	"app/base/core"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatus(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configureStatus()

	assert.Equal(t, "Installable", STATUS[INSTALLABLE])
	assert.Equal(t, "Applicable", STATUS[APPLICABLE])
}
