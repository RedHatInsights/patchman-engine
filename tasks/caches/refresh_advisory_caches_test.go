package caches

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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

	var wg sync.WaitGroup
	refreshAdvisoryCachesPerAccounts(&wg)
	wg.Wait()

	assert.Equal(t, 2, database.PluckInt(database.Db.Table("advisory_account_data").
		Where("advisory_id = 1 AND rh_account_id = 2"), "systems_affected"))
	assert.Equal(t, 1, database.PluckInt(database.Db.Table("advisory_account_data").
		Where("advisory_id = 2 AND rh_account_id = 1"), "systems_affected"))
	assert.Equal(t, 1, database.PluckInt(database.Db.Table("advisory_account_data").
		Where("advisory_id = 3 AND rh_account_id = 1"), "systems_affected"))
}
