package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRefreshAdvisoryCachesPerAccounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	// set wrong numbers of caches
	assert.Nil(t, database.Db.Model(&models.AdvisoryAccountData{}).
		Where("advisory_id = 1 AND rh_account_id = 2").Update("systems_affected", 5).Error)
	assert.Nil(t, database.Db.Model(&models.AdvisoryAccountData{}).
		Where("advisory_id = 2 AND rh_account_id = 1").Update("systems_affected", 3).Error)
	assert.Nil(t, database.Db.Model(&models.AdvisoryAccountData{}).
		Where("advisory_id = 3 AND rh_account_id = 1").Update("systems_affected", 8).Error)

	refreshAdvisoryCachesPerAccounts(0)

	assert.Equal(t, 2, database.PluckInt(database.Db.Table("advisory_account_data").
		Where("advisory_id = 1 AND rh_account_id = 2"), "systems_affected"))
	assert.Equal(t, 1, database.PluckInt(database.Db.Table("advisory_account_data").
		Where("advisory_id = 2 AND rh_account_id = 1"), "systems_affected"))
	assert.Equal(t, 1, database.PluckInt(database.Db.Table("advisory_account_data").
		Where("advisory_id = 3 AND rh_account_id = 1"), "systems_affected"))
}
