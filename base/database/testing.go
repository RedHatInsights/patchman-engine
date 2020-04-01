package database

import (
	"app/base/models"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func DebugWithCachesCheck(part string, fun func()) {
	fun()
	validAfter, err := CheckCachesValid()
	if err != nil {
		utils.Log("error", err).Error("Could not check validity of caches")
	}

	if !validAfter {
		utils.Log("part", part).Panic("Cache mismatch created")
	}
}

type key struct {
	AccountID  int
	AdvisoryID int
}

type advisoryCount struct {
	RhAccountID int
	AdvisoryID  int
	Count       int
}

func CheckCachesValid() (bool, error) {
	valid := true
	var aad []models.AdvisoryAccountData

	tx := Db.Begin()
	err := tx.Set("gorm:query_option", "FOR SHARE OF advisory_account_data").
		Order("rh_account_id, advisory_id").Find(&aad).Error
	if err != nil {
		return false, err
	}
	var counts []advisoryCount

	err = tx.Select("sp.rh_account_id, sa.advisory_id, count(system_id)").
		Table("system_advisories sa").
		Joins("JOIN system_platform sp on sa.system_id = sp.id").
		Where("sa.when_patched is null AND sp.stale = false AND sp.last_evaluation is not null").
		Order("sp.rh_account_id,  sa.advisory_id").
		Group("sp.rh_account_id,  sa.advisory_id").
		Find(&counts).Error
	if err != nil {
		return false, err
	}

	cached := map[key]int{}
	calculated := map[key]int{}

	for _, val := range aad {
		cached[key{val.RhAccountID, val.AdvisoryID}] = val.SystemsAffected
	}
	for _, val := range counts {
		calculated[key{val.RhAccountID, val.AdvisoryID}] = val.Count
	}

	for key, cachedCount := range cached {
		calcCount := calculated[key]

		if cachedCount != calcCount {
			utils.Log("advisory_id", key.AdvisoryID, "account_id", key.AccountID,
				"cached", cachedCount, "calculated", calcCount).Error("Cached counts mismatch")
			valid = false
		}
	}

	for key, calcCount := range calculated {
		cachedCount := calculated[key]

		if cachedCount != calcCount {
			utils.Log("advisory_id", key.AdvisoryID, "account_id", key.AccountID,
				"cached", cachedCount, "calculated", calcCount).Error("Cached counts mismatch")
			valid = false
		}
	}
	tx.Commit()
	tx.RollbackUnlessCommitted()
	return valid, nil
}

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

	sum := 0
	for _, item := range advisoryAccountData {
		sum += item.SystemsAffected
	}
	// covers both cases, when we have advisory_account_data stored with 0 systems_affected, and when we delete it
	assert.Equal(t, systemsAffected*len(advisoryIDs), sum, "sum of systems_affected does not match")
}
