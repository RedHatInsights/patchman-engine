package database

import (
	"app/base"
	"app/base/models"
	"app/base/utils"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func DebugWithCachesCheck(part string, fun func()) {
	fun()
	validAfter, err := CheckCachesValidRet()
	if err != nil {
		utils.Log("error", err).Panic("Could not check validity of caches")
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

func CheckCachesValidRet() (bool, error) {
	valid := true
	var aad []models.AdvisoryAccountData

	tx := Db.WithContext(base.Context).Begin()
	defer tx.Rollback()
	err := tx.Set("gorm:query_option", "FOR SHARE OF advisory_account_data").
		Order("rh_account_id, advisory_id").Find(&aad).Error
	if err != nil {
		return false, err
	}
	var counts []advisoryCount

	err = tx.Select("sp.rh_account_id, sa.advisory_id, count(*)").
		Table("system_advisories sa").
		Joins("JOIN system_platform sp ON sa.rh_account_id = sp.rh_account_id AND sa.system_id = sp.id").
		Where("sa.when_patched IS NULL AND sp.stale = false AND sp.last_evaluation IS NOT NULL").
		Order("sp.rh_account_id, sa.advisory_id").
		Group("sp.rh_account_id, sa.advisory_id").
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
	return valid, nil
}

func CheckCachesValid(t *testing.T) {
	valid, err := CheckCachesValidRet()
	assert.Nil(t, err)
	assert.True(t, valid)
}

func CheckAdvisoriesInDB(t *testing.T, advisories []string) []int {
	var advisoryIDs []int
	err := Db.Model(models.AdvisoryMetadata{}).Where("name IN (?)", advisories).
		Pluck("id", &advisoryIDs).Error
	assert.Nil(t, err)
	assert.Equal(t, len(advisories), len(advisoryIDs))
	return advisoryIDs
}

func CheckThirdPartyRepos(t *testing.T, repoNames []string, thirdParty bool) {
	var reposObjs []models.Repo
	assert.Nil(t, Db.Where("name IN (?)", repoNames).Find(&reposObjs).Error)
	assert.Equal(t, len(reposObjs), len(repoNames), "loaded repos count match")
	for _, reposObj := range reposObjs {
		assert.Equal(t, thirdParty, reposObj.ThirdParty,
			fmt.Sprintf("thirdParty flag for '%s'", reposObj.Name))
	}
}

func CheckPackagesNamesInDB(t *testing.T, packageNames ...string) {
	var count int64
	assert.Nil(t, Db.Model(&models.PackageName{}).Where("name in (?)", packageNames).Count(&count).Error)
	assert.Equal(t, int64(len(packageNames)), count)
}

func CheckSystemJustEvaluated(t *testing.T, inventoryID string, nAll, nEnh, nBug, nSec, nInstall, nUpdate int,
	thirdParty bool) {
	var system models.SystemPlatform
	assert.Nil(t, Db.Where("inventory_id = ?::uuid", inventoryID).First(&system).Error)
	assert.NotNil(t, system.LastEvaluation)
	assert.True(t, system.LastEvaluation.After(time.Now().Add(-time.Second)))
	assert.Equal(t, nAll, system.AdvisoryCountCache)
	assert.Equal(t, nEnh, system.AdvisoryEnhCountCache)
	assert.Equal(t, nBug, system.AdvisoryBugCountCache)
	assert.Equal(t, nSec, system.AdvisorySecCountCache)
	assert.Equal(t, nInstall, system.PackagesInstalled)
	assert.Equal(t, nUpdate, system.PackagesUpdatable)
	assert.Equal(t, thirdParty, system.ThirdParty)
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

func CreateReportedAdvisories(reportedAdvisories ...string) map[string]bool {
	reportedAdvisoriesMap := map[string]bool{}
	for _, adv := range reportedAdvisories {
		reportedAdvisoriesMap[adv] = true
	}
	return reportedAdvisoriesMap
}

func CreateStoredAdvisories(advisoryPatched map[int]*time.Time) map[string]models.SystemAdvisories {
	systemAdvisoriesMap := map[string]models.SystemAdvisories{}
	for advisoryID, patched := range advisoryPatched {
		systemAdvisoriesMap["ER-"+strconv.Itoa(advisoryID)] = models.SystemAdvisories{
			WhenPatched: patched,
			AdvisoryID:  advisoryID}
	}
	return systemAdvisoriesMap
}

func CreateSystemAdvisories(t *testing.T, rhAccountID int, systemID int, advisoryIDs []int,
	whenPatched *time.Time) {
	for _, advisoryID := range advisoryIDs {
		err := Db.Create(&models.SystemAdvisories{
			RhAccountID: rhAccountID, SystemID: systemID, AdvisoryID: advisoryID, WhenPatched: whenPatched}).Error
		assert.Nil(t, err)
	}
	CheckSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, whenPatched)
}

func CreateAdvisoryAccountData(t *testing.T, rhAccountID int, advisoryIDs []int,
	systemsAffected int) {
	for _, advisoryID := range advisoryIDs {
		err := Db.Create(&models.AdvisoryAccountData{
			AdvisoryID: advisoryID, RhAccountID: rhAccountID, SystemsAffected: systemsAffected}).Error
		assert.Nil(t, err)
	}
	CheckAdvisoriesAccountData(t, rhAccountID, advisoryIDs, systemsAffected)
}

func CreateSystemRepos(t *testing.T, rhAccountID int, systemID int, repoIDs []int) {
	for _, repoID := range repoIDs {
		assert.Nil(t, Db.Create(&models.SystemRepo{RhAccountID: rhAccountID, SystemID: systemID, RepoID: repoID}).Error)
	}
	CheckSystemRepos(t, rhAccountID, systemID, repoIDs)
}

func CheckSystemAdvisoriesWhenPatched(t *testing.T, systemID int, advisoryIDs []int,
	whenPatched *time.Time) {
	var systemAdvisories []models.SystemAdvisories
	err := Db.Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
		Find(&systemAdvisories).Error
	assert.Nil(t, err)
	assert.Equal(t, len(advisoryIDs), len(systemAdvisories))
	for _, systemAdvisory := range systemAdvisories {
		assert.NotNil(t, systemAdvisory.FirstReported)
		if whenPatched == nil {
			assert.Nil(t, systemAdvisory.WhenPatched)
		} else {
			assert.Equal(t, systemAdvisory.WhenPatched.String(), whenPatched.String())
		}
	}
}

func CheckEVRAsInDB(t *testing.T, nExpected int, evras ...string) {
	var packages []models.Package
	assert.Nil(t, Db.Where("evra IN (?)", evras).Find(&packages).Error)
	assert.Equal(t, nExpected, len(packages))
}

func CheckSystemPackages(t *testing.T, systemID int, nExpected int, packageIDs ...int) {
	var systemPackages []models.SystemPackage
	query := Db.Where("system_id = ?", systemID)
	if len(packageIDs) > 0 {
		query = query.Where("package_id IN (?)", packageIDs)
	}
	assert.Nil(t, query.Find(&systemPackages).Error)
	assert.Equal(t, nExpected, len(systemPackages))
}

func CheckSystemRepos(t *testing.T, rhAccountID int, systemID int, repoIDs []int) {
	var systemRepos []models.SystemRepo
	err := Db.Where("rh_account_id = ? AND system_id = ? AND repo_id IN (?)",
		rhAccountID, systemID, repoIDs).
		Find(&systemRepos).Error
	assert.Nil(t, err)
	assert.Equal(t, len(repoIDs), len(systemRepos))
}

func DeleteSystemAdvisories(t *testing.T, systemID int, advisoryIDs []int) {
	query := Db.Model(&models.SystemAdvisories{}).Where("system_id = ? AND advisory_id IN (?)",
		systemID, advisoryIDs)
	assert.Nil(t, query.Delete(&models.SystemAdvisories{}).Error)

	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
	assert.Nil(t, Db.Exec("SELECT * FROM update_system_caches(?)", systemID).Error)
}

func DeleteAdvisoryAccountData(t *testing.T, rhAccountID int, advisoryIDs []int) {
	query := Db.Model(&models.AdvisoryAccountData{}).Where("rh_account_id = ? AND advisory_id IN (?)",
		rhAccountID, advisoryIDs)
	assert.Nil(t, query.Delete(&models.AdvisoryAccountData{}).Error)

	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func DeleteSystemPackages(t *testing.T, systemID int, pkgIDs ...int) {
	query := Db.Model(&models.SystemPackage{}).Where("system_id = ?", systemID)
	if len(pkgIDs) > 0 {
		query = query.Where("package_id in (?)", pkgIDs)
	}
	assert.Nil(t, query.Delete(&models.SystemPackage{}).Error)

	var count int64 // ensure deleted
	assert.Nil(t, query.Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func DeleteSystemRepos(t *testing.T, rhAccountID int, systemID int, repoIDs []int) {
	err := Db.Where("rh_account_id = ? AND system_id = ? AND repo_id IN (?)", rhAccountID, systemID, repoIDs).
		Delete(&models.SystemRepo{}).Error
	assert.Nil(t, err)
}

func DeleteNewlyAddedPackages(t *testing.T) {
	query := Db.Model(models.Package{}).Where("id >= 100")
	assert.Nil(t, query.Delete(models.Package{}).Error)
	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func DeleteNewlyAddedAdvisories(t *testing.T) {
	query := Db.Model(models.AdvisoryMetadata{}).Where("id >= 100")
	assert.Nil(t, query.Delete(models.AdvisoryMetadata{}).Error)
	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func UpdateSystemAdvisoriesWhenPatched(t *testing.T, systemID, accountID int, advisoryIDs []int,
	whenPatched *time.Time) {
	err := Db.Model(models.SystemAdvisories{}).
		Where("system_id = ?", systemID).
		Where("rh_account_id = ?", accountID).
		Where("advisory_id IN (?)", advisoryIDs).
		Update("when_patched", whenPatched).Error
	assert.Nil(t, err)
}
