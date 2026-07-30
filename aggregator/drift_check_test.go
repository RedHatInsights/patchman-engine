package aggregator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCheckAdvisoryDriftCountMismatch(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	assert.Nil(t, database.DB.Exec("SELECT refresh_advisory_caches(NULL, 1)").Error)
	assert.Nil(t, database.DB.Exec("SELECT backfill_account_advisory(1)").Error)
	defer database.DeleteAccountAdvisoryByAccount(t, 1)

	assert.Nil(t, database.DB.Model(&models.AccountAdvisory{}).
		Where("advisory_id = 1 AND rh_account_id = 1").
		Update("systems_installable", 999).Error)

	hook := utils.NewTestLogHook(log.WarnLevel)
	log.AddHook(hook)

	checkAdvisoryDrift(1, []int64{1})

	found := false
	for _, entry := range hook.LogEntries {
		if entry.Message == "drift check: count mismatch between legacy and new table" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected count mismatch warning")
}

func TestCheckAdvisoryDriftMissingFromNew(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	assert.Nil(t, database.DB.Exec("SELECT refresh_advisory_caches(NULL, 1)").Error)
	defer database.DeleteAccountAdvisoryByAccount(t, 1)

	hook := utils.NewTestLogHook(log.WarnLevel)
	log.AddHook(hook)

	checkAdvisoryDrift(1, []int64{1, 2})

	found := false
	for _, entry := range hook.LogEntries {
		if entry.Message == "drift check: advisory present in legacy table but missing from account_advisory" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing from new table warning")
}
