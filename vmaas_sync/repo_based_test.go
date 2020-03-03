package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetLastRepobasedEvalTms(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	ts, err := getLastRepobasedEvalTms()
	assert.Nil(t, err)
	assert.Equal(t, "2018-04-04 23:23:45 +0000 UTC", ts.String())
}
