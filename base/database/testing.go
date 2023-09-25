package database

import (
	"app/base"
	"app/base/models"
	"app/base/utils"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func DebugWithCachesCheck(part string, fun func()) {
	fun()
	validAfter, err := CheckCachesValidRet()
	if err != nil {
		utils.LogPanic("error", err, "Could not check validity of caches")
	}

	if !validAfter {
		utils.LogPanic("part", part, "Cache mismatch created")
	}
}

type key struct {
	AccountID  int
	AdvisoryID int64
}

type advisoryCount struct {
	RhAccountID        int
	AdvisoryID         int64
	SystemsInstallable int
	SystemsApplicable  int
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

	err = tx.Select("sp.rh_account_id, sa.advisory_id," +
		"count(*) filter (where sa.status_id = 0) as systems_installable," +
		"count(*) as systems_applicable").
		Table("system_advisories sa").
		Joins("JOIN system_platform sp ON sa.rh_account_id = sp.rh_account_id AND sa.system_id = sp.id").
		Where("sp.stale = false AND sp.last_evaluation IS NOT NULL").
		Order("sp.rh_account_id, sa.advisory_id").
		Group("sp.rh_account_id, sa.advisory_id").
		Find(&counts).Error
	if err != nil {
		return false, err
	}

	cached := make(map[key][]int, len(aad))
	calculated := make(map[key][]int, len(counts))

	for _, val := range aad {
		cached[key{val.RhAccountID, val.AdvisoryID}] = []int{val.SystemsInstallable, val.SystemsApplicable}
	}
	for _, val := range counts {
		calculated[key{val.RhAccountID, val.AdvisoryID}] = []int{val.SystemsInstallable, val.SystemsApplicable}
	}

	crossCheckCache := func(a, b map[key][]int) {
		for key, aCounts := range a {
			bCounts := b[key]
			if len(bCounts) == 0 {
				bCounts = []int{0, 0}
			}
			for i, msg := range []string{"installable", "applicable"} {
				if aCounts[i] != bCounts[i] {
					utils.LogError("advisory_id", key.AdvisoryID, "account_id", key.AccountID,
						"cached", aCounts[i], "calculated", bCounts[i], fmt.Sprintf("Cached %s counts mismatch", msg))
					valid = false
				}
			}
		}
	}
	crossCheckCache(cached, calculated)
	crossCheckCache(calculated, cached)

	tx.Commit()
	return valid, nil
}

func CheckCachesValid(t *testing.T) {
	valid, err := CheckCachesValidRet()
	assert.Nil(t, err)
	assert.True(t, valid)
}

func CheckAdvisoriesInDB(t *testing.T, advisories []string) []int64 {
	var advisoryIDs []int64
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

func CheckPackagesNamesInDB(t *testing.T, filter string, packageNames ...string) {
	var count int64
	query := Db.Model(&models.PackageName{}).Where("name in (?)", packageNames)
	if filter != "" {
		query = query.Where(filter)
	}
	assert.Nil(t, query.Count(&count).Error)
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

func CheckAdvisoriesAccountData(t *testing.T, rhAccountID int, advisoryIDs []int64, systemsInstallable int) {
	var advisoryAccountData []models.AdvisoryAccountData
	err := Db.Where("rh_account_id = ? AND advisory_id IN (?)", rhAccountID, advisoryIDs).
		Find(&advisoryAccountData).Error
	assert.Nil(t, err)

	sum := 0
	for _, item := range advisoryAccountData {
		sum += item.SystemsInstallable
	}
	// covers both cases, when we have advisory_account_data stored with 0 systems_installable, and when we delete it
	assert.Equal(t, systemsInstallable*len(advisoryIDs), sum, "sum of systems_installable does not match")
}

func CheckAdvisoriesAccountDataNotified(t *testing.T, rhAccountID int, advisoryIDs []int64, notified bool) {
	var advisoryAccountData []models.AdvisoryAccountData
	err := Db.Where("rh_account_id = ? AND advisory_id IN (?)", rhAccountID, advisoryIDs).
		Find(&advisoryAccountData).Error
	assert.Nil(t, err)

	for _, item := range advisoryAccountData {
		if notified {
			assert.NotNil(t, item.Notified)
		} else {
			assert.Nil(t, item.Notified)
		}
	}
}

func CheckSystemUpdatesCount(t *testing.T, accountID int, systemID int64) []int {
	var cnt []int
	assert.NoError(t, Db.Table("system_package spkg").
		Select("json_array_length(spkg.update_data::json) as len").
		Where("spkg.update_data is not null").
		Where("spkg.system_id = ? AND spkg.rh_account_id = ? ", systemID, accountID).
		Pluck("len", &cnt).Error)
	return cnt
}

func CreateReportedAdvisories(reportedAdvisories []string, status []int) map[string]int {
	reportedAdvisoriesMap := make(map[string]int, len(reportedAdvisories))
	for i, adv := range reportedAdvisories {
		reportedAdvisoriesMap[adv] = status[i]
	}
	return reportedAdvisoriesMap
}

func CreateStoredAdvisories(advisoryPatched []int64) map[string]models.SystemAdvisories {
	systemAdvisoriesMap := make(map[string]models.SystemAdvisories, len(advisoryPatched))
	for _, advisoryID := range advisoryPatched {
		systemAdvisoriesMap["ER-"+strconv.FormatInt(advisoryID, 10)] = models.SystemAdvisories{
			AdvisoryID: advisoryID}
	}
	return systemAdvisoriesMap
}

func CreateSystemAdvisories(t *testing.T, rhAccountID int, systemID int64, advisoryIDs []int64) {
	for _, advisoryID := range advisoryIDs {
		err := Db.Create(&models.SystemAdvisories{
			RhAccountID: rhAccountID, SystemID: systemID, AdvisoryID: advisoryID, StatusID: 0}).Error
		assert.Nil(t, err)
	}
	CheckSystemAdvisories(t, systemID, advisoryIDs)
}

func CreateAdvisoryAccountData(t *testing.T, rhAccountID int, advisoryIDs []int64,
	systemsInstallable int) {
	for _, advisoryID := range advisoryIDs {
		err := Db.Create(&models.AdvisoryAccountData{
			AdvisoryID: advisoryID, RhAccountID: rhAccountID, SystemsInstallable: systemsInstallable,
			// create same number of applicable and installable systems because installable is subset of applicable
			SystemsApplicable: systemsInstallable}).Error
		assert.Nil(t, err)
	}
	CheckAdvisoriesAccountData(t, rhAccountID, advisoryIDs, systemsInstallable)
}

func CreateSystemRepos(t *testing.T, rhAccountID int, systemID int64, repoIDs []int64) {
	for _, repoID := range repoIDs {
		assert.Nil(t, Db.Create(&models.SystemRepo{RhAccountID: int64(rhAccountID),
			SystemID: systemID, RepoID: repoID}).Error)
	}
	CheckSystemRepos(t, rhAccountID, systemID, repoIDs)
}

func CheckSystemAdvisories(t *testing.T, systemID int64, advisoryIDs []int64) {
	var systemAdvisories []models.SystemAdvisories
	err := Db.Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
		Find(&systemAdvisories).Error
	assert.Nil(t, err)
	assert.Equal(t, len(advisoryIDs), len(systemAdvisories))
	for _, systemAdvisory := range systemAdvisories {
		assert.NotNil(t, systemAdvisory.FirstReported)
	}
}

func CheckEVRAsInDB(t *testing.T, nExpected int, evras ...string) {
	var packages []models.Package
	assert.Nil(t, Db.Where("evra IN (?)", evras).Find(&packages).Error)
	assert.Equal(t, nExpected, len(packages))
}

func CheckEVRAsInDBSynced(t *testing.T, nExpected int, synced bool, evras ...string) {
	var packages []models.Package
	assert.Nil(t, Db.Where("evra IN (?)", evras).Find(&packages).Error)
	assert.Equal(t, nExpected, len(packages))
	for _, pkg := range packages {
		assert.Equal(t, synced, pkg.Synced)
	}
}

func CheckSystemPackages(t *testing.T, accountID int, systemID int64, nExpected int, packageIDs ...int64) {
	var systemPackages []models.SystemPackage
	query := Db.Where("rh_account_id = ? AND system_id = ?", accountID, systemID)
	if len(packageIDs) > 0 {
		query = query.Where("package_id IN (?)", packageIDs)
	}
	assert.Nil(t, query.Find(&systemPackages).Error)
	assert.Equal(t, nExpected, len(systemPackages))
}

func CheckSystemRepos(t *testing.T, rhAccountID int, systemID int64, repoIDs []int64) {
	var systemRepos []models.SystemRepo
	err := Db.Where("rh_account_id = ? AND system_id = ? AND repo_id IN (?)",
		rhAccountID, systemID, repoIDs).
		Find(&systemRepos).Error
	assert.Nil(t, err)
	assert.Equal(t, len(repoIDs), len(systemRepos))
}

func DeleteSystemAdvisories(t *testing.T, systemID int64, advisoryIDs []int64) {
	query := Db.Model(&models.SystemAdvisories{}).Where("system_id = ? AND advisory_id IN (?)",
		systemID, advisoryIDs)
	assert.Nil(t, query.Delete(&models.SystemAdvisories{}).Error)

	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
	assert.Nil(t, Db.Exec("SELECT * FROM update_system_caches(?)", systemID).Error)
}

func DeleteAdvisoryAccountData(t *testing.T, rhAccountID int, advisoryIDs []int64) {
	query := Db.Model(&models.AdvisoryAccountData{}).Where("rh_account_id = ? AND advisory_id IN (?)",
		rhAccountID, advisoryIDs)
	assert.Nil(t, query.Delete(&models.AdvisoryAccountData{}).Error)

	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func DeleteSystemPackages(t *testing.T, accountID int, systemID int64, pkgIDs ...int64) {
	query := Db.Model(&models.SystemPackage{}).Where("rh_account_id = ? AND system_id = ?", accountID, systemID)
	if len(pkgIDs) > 0 {
		query = query.Where("package_id in (?)", pkgIDs)
	}
	assert.Nil(t, query.Delete(&models.SystemPackage{}).Error)

	var count int64 // ensure deleted
	assert.Nil(t, query.Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func DeleteSystemRepos(t *testing.T, rhAccountID int, systemID int64, repoIDs []int64) {
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

func CreateBaselineWithConfig(t *testing.T, name string, inventoryIDs []string,
	configBytes []byte, description *string) int64 {
	if name == "" {
		name = "temporary_baseline"
	}

	temporaryBaseline := &models.Baseline{
		RhAccountID: 1, Name: name, Config: configBytes, Description: description,
	}

	tx := Db.WithContext(base.Context).Begin()
	defer tx.Rollback()

	if err := tx.Create(temporaryBaseline).Error; err != nil {
		assert.Nil(t, err)
	}

	err := tx.Model(models.SystemPlatform{}).
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Where("rh_account_id = (?) AND inventory_id::text IN (?)", 1, inventoryIDs).
		Update("baseline_id", temporaryBaseline.ID).Error
	assert.Nil(t, err)
	err = tx.Commit().Error
	assert.Nil(t, err)
	return temporaryBaseline.ID
}

func CreateBaseline(t *testing.T, name string, inventoryIDs []string, description *string) int64 {
	configBytes := []byte(`{"to_time": "2021-01-01T12:00:00-04:00"}`)
	baselineID := CreateBaselineWithConfig(t, name, inventoryIDs, configBytes, description)
	return baselineID
}

func DeleteBaseline(t *testing.T, baselineID int64) {
	tx := Db.WithContext(base.Context).Begin()
	defer tx.Rollback()

	err := tx.Model(models.SystemPlatform{}).
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Where("rh_account_id = (?) AND baseline_id = (?)", 1, baselineID).
		Update("baseline_id", nil).Error

	assert.Nil(t, err)

	err = tx.Where(models.Baseline{ID: baselineID, RhAccountID: 1}).Delete(&models.Baseline{}).Error
	assert.Nil(t, err)

	err = tx.Commit().Error
	assert.Nil(t, err)
}

func CheckBaseline(t *testing.T, baselineID int64, inventoryIDs []string, config, name string, description *string) {
	type Baseline struct {
		ID          int64   `query:"bl.id" gorm:"column:id"`
		Name        string  `json:"name" query:"bl.name" gorm:"column:name"`
		Config      string  `json:"config" query:"bl.config" gorm:"column:config"`
		Description *string `json:"description" query:"bl.description" gorm:"column:description"`
	}

	type Associations struct {
		ID string `json:"system" query:"id"`
	}

	var associations []Associations
	var baseline Baseline

	err := Db.Table("system_platform as sp").Select("sp.inventory_id as id").
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Where("sp.rh_account_id = (?) AND sp.baseline_id = (?)", 1, baselineID).Order("id").Find(&associations).Error

	assert.Nil(t, err)

	err = Db.Table("baseline as bl").
		Select("bl.id, bl.name, bl.config, bl.description").
		Where("bl.rh_account_id = (?) AND bl.id = (?)", 1, baselineID).Find(&baseline).Error

	assert.Nil(t, err)

	assert.Equal(t, baselineID, baseline.ID)
	assert.Equal(t, name, baseline.Name)
	assert.Equal(t, config, baseline.Config)
	if description == nil {
		assert.Equal(t, description, baseline.Description)
	} else {
		assert.Equal(t, *description, *baseline.Description)
	}

	if len(inventoryIDs) == 0 {
		assert.Equal(t, len(associations), 0)
	} else {
		for index, inventoryID := range inventoryIDs {
			assert.Equal(t, associations[index].ID, inventoryID)
		}
	}
}

func CheckBaselineDeleted(t *testing.T, baselineID int64) {
	var cntBaselines int64
	assert.Nil(t, Db.Model(&models.Baseline{}).Where("id = ?", baselineID).Count(&cntBaselines).Error)
	assert.Equal(t, 0, int(cntBaselines))

	var cntSystems int64
	assert.Nil(t, Db.Model(&models.SystemPlatform{}).Where("baseline_id = ?", baselineID).Count(&cntSystems).Error)
	assert.Equal(t, 0, int(cntSystems))
}

func GetAllSystems(t *testing.T) (systems []*models.SystemPlatform) {
	assert.Nil(t, Db.Model(&models.SystemPlatform{}).Order("rh_account_id").Scan(&systems).Error)
	return systems
}
