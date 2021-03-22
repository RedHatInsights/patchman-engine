package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"context"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var testDate, _ = time.Parse(time.RFC3339, "2020-01-01T01-01-01")

func TestInit(t *testing.T) {
	utils.TestLoadEnv("conf/evaluator_common.env", "conf/evaluator_upload.env")
}

func TestVMaaSGetUpdates(t *testing.T) {
	utils.SkipWithoutPlatform(t)

	configure()
	vmaasData := getVMaaSUpdates(t)
	for k, v := range vmaasData.UpdateList {
		if strings.HasPrefix(k, "firefox") {
			assert.Equal(t, 2, len(v.AvailableUpdates))
		} else if strings.HasPrefix(k, "kernel") {
			assert.Equal(t, 1, len(v.AvailableUpdates))
		}
	}
}

func TestGetReportedAdvisories1(t *testing.T) {
	utils.SkipWithoutPlatform(t)

	configure()
	vmaasData := getVMaaSUpdates(t)
	advisories := getReportedAdvisories(vmaasData)
	assert.Equal(t, 3, len(advisories))
}

func TestGetReportedAdvisories2(t *testing.T) {
	vmaasData := vmaas.UpdatesV2Response{
		UpdateList: map[string]vmaas.UpdatesV2ResponseUpdateList{
			"pkg-a": {AvailableUpdates: []vmaas.UpdatesResponseAvailableUpdates{{Erratum: "ER1"}, {Erratum: "ER2"}}},
			"pkg-b": {AvailableUpdates: []vmaas.UpdatesResponseAvailableUpdates{{Erratum: "ER2"}, {Erratum: "ER3"}}},
			"pkg-c": {AvailableUpdates: []vmaas.UpdatesResponseAvailableUpdates{{Erratum: "ER3"}, {Erratum: "ER4"}}},
		},
	}
	advisories := getReportedAdvisories(vmaasData)
	assert.Equal(t, 4, len(advisories))
}

func TestGetStoredAdvisoriesMap(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemAdvisories, err := getStoredAdvisoriesMap(database.Db, 1, 1)
	assert.Nil(t, err)
	assert.NotNil(t, systemAdvisories)
	assert.Equal(t, 9, len(systemAdvisories))
	assert.Equal(t, "RH-1", (systemAdvisories)["RH-1"].Advisory.Name)
	assert.Equal(t, "adv-1-des", (systemAdvisories)["RH-1"].Advisory.Description)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", (systemAdvisories)["RH-1"].Advisory.PublicDate.String())
}

func TestGetNewAndUnpatchedAdvisories(t *testing.T) {
	stored := createStoredAdvisories(map[int]*time.Time{1: &testDate, 2: nil, 3: nil})
	reported := createReportedAdvisories("ER-1", "ER-3", "ER-4")
	news, unpatched := getNewAndUnpatchedAdvisories(reported, stored)
	assert.Equal(t, 1, len(news))
	assert.Equal(t, "ER-4", news[0])
	assert.Equal(t, 1, len(unpatched))
	assert.Equal(t, 1, unpatched[0])
}

func TestGetPatchedAdvisories(t *testing.T) {
	stored := createStoredAdvisories(map[int]*time.Time{1: &testDate, 2: nil, 3: nil})
	reported := createReportedAdvisories("ER-3", "ER-4")
	patched := getPatchedAdvisories(reported, stored)
	assert.Equal(t, 1, len(patched))
	assert.Equal(t, 2, patched[0])
}

func updateSystemAdvisoriesWhenPatched(systemID, accountID int, advisoryIDs []int, whenPatched *time.Time) error {
	err := database.Db.Model(models.SystemAdvisories{}).
		Where("system_id = ?", systemID).
		Where("rh_account_id = ?", accountID).
		Where("advisory_id IN (?)", advisoryIDs).
		Update("when_patched", whenPatched).Error
	if err != nil {
		return err
	}
	return nil
}

func TestUpdatePatchedSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	system := models.SystemPlatform{ID: 12, RhAccountID: 3}
	advisoryIDs := []int{2, 3, 4}
	createSystemAdvisories(t, system.RhAccountID, system.ID, advisoryIDs, nil)
	createAdvisoryAccountData(t, system.RhAccountID, advisoryIDs, 1)

	err := updateSystemAdvisoriesWhenPatched(system.ID, system.RhAccountID, advisoryIDs, &testDate)
	assert.NoError(t, err)
	// Update as-if the advisories had become patched
	err = updateAdvisoryAccountDatas(database.Db, &system, advisoryIDs, []int{})
	assert.NoError(t, err)

	checkSystemAdvisoriesWhenPatched(t, system.ID, advisoryIDs, &testDate)
	database.CheckAdvisoriesAccountData(t, system.RhAccountID, advisoryIDs, 0)

	err = updateSystemAdvisoriesWhenPatched(system.ID, system.RhAccountID, advisoryIDs, nil)
	assert.NoError(t, err)
	// Update as-if the advisories had become unpatched
	err = updateAdvisoryAccountDatas(database.Db, &system, []int{}, advisoryIDs)
	assert.NoError(t, err)

	database.CheckAdvisoriesAccountData(t, system.RhAccountID, advisoryIDs, 1)

	deleteSystemAdvisories(t, system.ID, advisoryIDs)
	deleteAdvisoryAccountData(t, system.RhAccountID, advisoryIDs)
}

func TestGetAdvisoriesFromDB(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	advisories := []string{"ER-1", "RH-1", "ER-2", "RH-2"}
	advisoryIDs, err := getAdvisoriesFromDB(database.Db, advisories)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(advisoryIDs))
}

func TestEnsureSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	rhAccountID := 1
	systemID := 2
	advisoryIDs := []int{2, 3, 4}
	err := ensureSystemAdvisories(database.Db, rhAccountID, systemID, advisoryIDs)
	assert.Nil(t, err)
	checkSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, nil)

	deleteSystemAdvisories(t, systemID, advisoryIDs)
}

// nolint: funlen
func TestEvaluate(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()

	configure()
	mockWriter := utils.MockKafkaWriter{}
	remediationsPublisher = &mockWriter

	systemID := 12
	rhAccountID := 3
	expectedAddedAdvisories := []string{"RH-1", "RH-2"}
	expectedAdvisoryIDs := []int{1, 2}       // advisories expected to be paired to the system after evaluation
	oldSystemAdvisoryIDs := []int{1, 3, 4}   // old advisories paired with the system
	patchingSystemAdvisoryIDs := []int{3, 4} // these advisories should be patched for the system
	expectedPackageIDs := []int{1, 2}
	systemRepoIDs := []int{1, 2}

	deleteSystemAdvisories(t, systemID, expectedAdvisoryIDs)
	deleteSystemAdvisories(t, systemID, patchingSystemAdvisoryIDs)
	deleteAdvisoryAccountData(t, rhAccountID, expectedAdvisoryIDs)
	deleteAdvisoryAccountData(t, rhAccountID, patchingSystemAdvisoryIDs)
	deleteSystemPackages(t, systemID, expectedPackageIDs)
	deleteSystemRepos(t, rhAccountID, systemID, systemRepoIDs)
	createSystemAdvisories(t, rhAccountID, systemID, oldSystemAdvisoryIDs, nil)
	createAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs, 1)
	createSystemRepos(t, rhAccountID, systemID, systemRepoIDs)
	database.CheckCachesValid(t)

	// do evaluate the system
	err := evaluateHandler(mqueue.PlatformEvent{
		SystemIDs: []string{"00000000-0000-0000-0000-000000000012", "00000000-0000-0000-0000-000000000011"},
		AccountID: rhAccountID})
	assert.NoError(t, err)

	advisoryIDs := database.CheckAdvisoriesInDB(t, expectedAddedAdvisories)
	checkSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, nil)
	checkSystemPackages(t, systemID, expectedPackageIDs)
	database.CheckSystemJustEvaluated(t, "00000000-0000-0000-0000-000000000012", 2, 1, 1,
		0, 2, 2, false)
	database.CheckCachesValid(t)

	// test evaluation with third party repos
	thirdPartySystemRepoIDs := []int{1, 2, 4}
	deleteSystemRepos(t, rhAccountID, systemID, systemRepoIDs)
	createSystemRepos(t, rhAccountID, systemID, thirdPartySystemRepoIDs)
	err = evaluateHandler(mqueue.PlatformEvent{
		SystemIDs: []string{"00000000-0000-0000-0000-000000000012"},
		AccountID: rhAccountID})
	assert.NoError(t, err)
	database.CheckSystemJustEvaluated(t, "00000000-0000-0000-0000-000000000012", 2, 1, 1,
		0, 2, 2, true)

	deleteSystemAdvisories(t, systemID, advisoryIDs)
	deleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
	deleteAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs)
	deleteSystemRepos(t, rhAccountID, systemID, thirdPartySystemRepoIDs)

	assert.Equal(t, 2, len(mockWriter.Messages))
}

func getVMaaSUpdates(t *testing.T) vmaas.UpdatesV2Response {
	vmaasCallArgs := vmaas.AppUpdatesHandlerV3PostPostOpts{}
	vmaasData, resp, err := vmaasClient.DefaultApi.AppUpdatesHandlerV3PostPost(context.Background(), &vmaasCallArgs)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Nil(t, resp.Body.Close())
	return vmaasData
}

func createReportedAdvisories(reportedAdvisories ...string) map[string]bool {
	reportedAdvisoriesMap := map[string]bool{}
	for _, adv := range reportedAdvisories {
		reportedAdvisoriesMap[adv] = true
	}
	return reportedAdvisoriesMap
}

func createStoredAdvisories(advisoryPatched map[int]*time.Time) map[string]models.SystemAdvisories {
	systemAdvisoriesMap := map[string]models.SystemAdvisories{}
	for advisoryID, patched := range advisoryPatched {
		systemAdvisoriesMap["ER-"+strconv.Itoa(advisoryID)] = models.SystemAdvisories{
			WhenPatched: patched,
			AdvisoryID:  advisoryID}
	}
	return systemAdvisoriesMap
}

func createSystemAdvisories(t *testing.T, rhAccountID int, systemID int, advisoryIDs []int,
	whenPatched *time.Time) {
	for _, advisoryID := range advisoryIDs {
		err := database.Db.Create(&models.SystemAdvisories{
			RhAccountID: rhAccountID, SystemID: systemID, AdvisoryID: advisoryID, WhenPatched: whenPatched}).Error
		assert.Nil(t, err)
	}
	checkSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, whenPatched)
}

func createAdvisoryAccountData(t *testing.T, rhAccountID int, advisoryIDs []int,
	systemsAffected int) {
	for _, advisoryID := range advisoryIDs {
		err := database.Db.Create(&models.AdvisoryAccountData{
			AdvisoryID: advisoryID, RhAccountID: rhAccountID, SystemsAffected: systemsAffected}).Error
		assert.Nil(t, err)
	}
	database.CheckAdvisoriesAccountData(t, rhAccountID, advisoryIDs, systemsAffected)
}

func createSystemRepos(t *testing.T, rhAccountID int, systemID int, repoIDs []int) {
	for _, repoID := range repoIDs {
		err := database.Db.Create(&models.SystemRepo{
			RhAccountID: rhAccountID, SystemID: systemID, RepoID: repoID}).Error
		assert.Nil(t, err)
	}
	checkSystemRepos(t, rhAccountID, systemID, repoIDs)
}

func checkSystemAdvisoriesWhenPatched(t *testing.T, systemID int, advisoryIDs []int,
	whenPatched *time.Time) {
	var systemAdvisories []models.SystemAdvisories
	err := database.Db.Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
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

func checkSystemPackages(t *testing.T, systemID int, packageIDs []int) {
	var systemPackages []models.SystemPackage
	err := database.Db.Where("system_id = ? AND package_id IN (?)", systemID, packageIDs).
		Find(&systemPackages).Error
	assert.Nil(t, err)
	assert.Equal(t, len(packageIDs), len(systemPackages))
}

func checkSystemRepos(t *testing.T, rhAccountID int, systemID int, repoIDs []int) {
	var systemRepos []models.SystemRepo
	err := database.Db.Where("rh_account_id = ? AND system_id = ? AND repo_id IN (?)",
		rhAccountID, systemID, repoIDs).
		Find(&systemRepos).Error
	assert.Nil(t, err)
	assert.Equal(t, len(repoIDs), len(systemRepos))
}

func deleteSystemAdvisories(t *testing.T, systemID int, advisoryIDs []int) {
	err := database.Db.Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
		Delete(&models.SystemAdvisories{}).Error
	assert.Nil(t, err)

	var systemAdvisories []models.SystemAdvisories
	err = database.Db.Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
		Find(&systemAdvisories).Error
	assert.Nil(t, err)
	assert.Equal(t, 0, len(systemAdvisories))
	assert.Nil(t, database.Db.Exec("SELECT * FROM update_system_caches(?)", systemID).Error)
}

func deleteAdvisoryAccountData(t *testing.T, rhAccountID int, advisoryIDs []int) {
	err := database.Db.Where("rh_account_id = ? AND advisory_id IN (?)", rhAccountID, advisoryIDs).
		Delete(&models.AdvisoryAccountData{}).Error
	assert.Nil(t, err)

	var advisoryAccountData []models.AdvisoryAccountData
	err = database.Db.Where("rh_account_id = ? AND advisory_id IN (?)", rhAccountID, advisoryIDs).
		Find(&advisoryAccountData).Error
	assert.Nil(t, err)
	assert.Equal(t, 0, len(advisoryAccountData))
}

func deleteSystemPackages(t *testing.T, systemID int, pkgIDs []int) {
	err := database.Db.Where("system_id = ? and package_id in(?)", systemID, pkgIDs).
		Delete(&models.SystemPackage{}).Error
	assert.Nil(t, err)
}

func deleteSystemRepos(t *testing.T, rhAccountID int, systemID int, repoIDs []int) {
	err := database.Db.Where("rh_account_id = ? AND system_id = ? AND repo_id IN (?)",
		rhAccountID, systemID, repoIDs).
		Delete(&models.SystemRepo{}).Error
	assert.Nil(t, err)
}

func TestRun(t *testing.T) {
	nReaders := 0
	wg := sync.WaitGroup{}
	wg.Add(1)
	run(&wg, mqueue.CreateCountedMockReader(&nReaders))
	utils.AssertWait(t, 10, func() bool {
		return nReaders == 8
	})
}
