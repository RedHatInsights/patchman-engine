package database

import (
	"app/base/models"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func CheckAdvisoriesInDb(t *testing.T, advisories []string) []int {
	var advisoriesObjs []models.AdvisoryMetadata
	err := Db.Where("name IN (?)", advisories).Find(&advisoriesObjs).Error
	assert.Nil(t, err)
	assert.Equal(t, len(advisoriesObjs), len(advisories))
	var ids []int //nolint:prealloc
	for _, advisoryObj := range advisoriesObjs {
		ids = append(ids, advisoryObj.ID)
	}
	return ids
}

func CheckSystemAdvisoriesFirstReportedGreater(t *testing.T, firstReported string, count int) {
	var systemAdvisories []models.SystemAdvisories
	err := Db.Where("first_reported > ?", firstReported).
		Find(&systemAdvisories).Error
	assert.Nil(t, err)
	assert.Equal(t, count, len(systemAdvisories))
}

func CheckSystemJustEvaluated(t *testing.T, inventoryID string, nAll, nEnh, nBug, nSec int) {
	var system models.SystemPlatform
	assert.Nil(t, Db.Where("inventory_id = ?", inventoryID).First(&system).Error)
	assert.NotNil(t, system.LastEvaluation)
	assert.True(t, system.LastEvaluation.After(time.Now().Add(-time.Second)))
	assert.Equal(t, nAll, system.AdvisoryCountCache)
	assert.Equal(t, nEnh, system.AdvisoryEnhCountCache)
	assert.Equal(t, nBug, system.AdvisoryBugCountCache)
	assert.Equal(t, nSec, system.AdvisorySecCountCache)
}

func CheckAdvisoriesAccountData(t *testing.T, rhAccountID int, advisoryIDs []int, systemsAffected int) {
	var advisoryAccountData []models.AdvisoryAccountData
	err := Db.Where("rh_account_id = ? AND advisory_id IN (?)", rhAccountID, advisoryIDs).
		Find(&advisoryAccountData).Error
	assert.Nil(t, err)
	assert.Equal(t, len(advisoryIDs), len(advisoryAccountData))
	for _, item := range advisoryAccountData {
		assert.Equal(t, systemsAffected, item.SystemsAffected)
	}
}
