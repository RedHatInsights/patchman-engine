package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"context"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strconv"
	"testing"
	"time"
)

var testDate, _ = time.Parse(time.RFC3339, "2020-01-01T01-01-01")

func TestVMaaSGetUpdates(t *testing.T) {
	utils.SkipWithoutPlatform(t)

	Configure()
	vmaasData := getVMaaSUpdates(t)
	assert.Equal(t, 2, len(vmaasData.UpdateList["firefox"].AvailableUpdates))
	assert.Equal(t, 1, len(vmaasData.UpdateList["kernel"].AvailableUpdates))
}

func TestGetReportedAdvisories1(t *testing.T) {
	utils.SkipWithoutPlatform(t)

	Configure()
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

	systemAdvisories, err := getStoredAdvisoriesMap(database.Db, 0)
	assert.Nil(t, err)
	assert.NotNil(t, systemAdvisories)
	assert.Equal(t, 8, len(*systemAdvisories))
	assert.Equal(t, "RH-1", (*systemAdvisories)["RH-1"].Advisory.Name)
	assert.Equal(t, "adv-1-des", (*systemAdvisories)["RH-1"].Advisory.Description)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", (*systemAdvisories)["RH-1"].Advisory.PublicDate.String())
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

func TestUpdatePatchedSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemID := 1
	rhAccountID := 0
	advisoryIDs := []int{2, 3, 4}
	createSystemAdvisories(t, systemID, advisoryIDs, nil)
	createAdvisoryAccountData(t, rhAccountID, advisoryIDs, 1)

	err := updateSystemAdvisoriesWhenPatched(database.Db, systemID, rhAccountID, advisoryIDs, &testDate)
	assert.Nil(t, err)
	checkSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, &testDate)
	database.CheckAdvisoriesAccountData(t, rhAccountID, advisoryIDs, 0)

	deleteSystemAdvisories(t, systemID, advisoryIDs)
	deleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
}

func TestUpdateUnpatchedSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemID := 1
	rhAccountID := 0
	advisoryIDs := []int{2, 3, 4}
	createSystemAdvisories(t, systemID, advisoryIDs, &testDate)
	createAdvisoryAccountData(t, rhAccountID, advisoryIDs, 1)

	err := updateSystemAdvisoriesWhenPatched(database.Db, systemID, rhAccountID, advisoryIDs, nil)
	assert.Nil(t, err)
	checkSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, nil)
	database.CheckAdvisoriesAccountData(t, rhAccountID, advisoryIDs, 2)

	deleteSystemAdvisories(t, systemID, advisoryIDs)
	deleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
}

func TestEnsureAdvisoriesInDb(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	advisories := []string{"ER-1", "RH-1", "ER-2", "RH-2"}
	advisoryIDs, nCreated, err := ensureAdvisoriesInDb(database.Db, advisories)
	assert.Nil(t, err)
	assert.Equal(t, 2, nCreated)
	assert.Equal(t, 4, len(*advisoryIDs))
	database.CheckAdvisoriesInDb(t, advisories)
	deleteAdvisories(t, []string{"ER-1", "ER-2"})
}

func TestAddNewSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemID := 1
	rhAccountID := 0
	advisoryIDs := []int{2, 3, 4}
	err := addNewSystemAdvisories(database.Db, systemID, rhAccountID, advisoryIDs)
	assert.Nil(t, err)
	checkSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, nil)

	deleteSystemAdvisories(t, systemID, advisoryIDs)
	deleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
}

func TestAddAndUpdateAccountAdvisoriesAffectedSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	rhAccountID := 2
	existingIDs := []int{1, 2}
	createAdvisoryAccountData(t, rhAccountID, existingIDs, 1)

	advisoryIDs := []int{1, 2, 3, 4}
	err := addAndUpdateAccountAdvisoriesAffectedSystems(database.Db, rhAccountID, advisoryIDs)
	assert.Nil(t, err)
	database.CheckAdvisoriesAccountData(t, rhAccountID, existingIDs, 2)
	database.CheckAdvisoriesAccountData(t, rhAccountID, []int{3, 4}, 1)

	deleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
}

func TestEvaluate(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()

	Configure()

	systemID := 11
	rhAccountID := 2
	expectedAddedAdvisories := []string{"ER1", "ER2", "ER3"}
	Evaluate(context.Background(), systemID, rhAccountID, vmaas.UpdatesV3Request{})
	advisoryIDs := database.CheckAdvisoriesInDb(t, expectedAddedAdvisories)

	checkSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, nil)
	database.CheckSystemJustEvaluated(t, "INV-11", 3, 0, 0, 0)

	deleteSystemAdvisories(t, systemID, advisoryIDs)
	deleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
	deleteAdvisories(t, expectedAddedAdvisories)
}

func getVMaaSUpdates(t *testing.T) vmaas.UpdatesV2Response {
	vmaasCallArgs := vmaas.AppUpdatesHandlerV3PostPostOpts{}
	vmaasData, resp, err := vmaasClient.UpdatesApi.AppUpdatesHandlerV3PostPost(context.Background(), &vmaasCallArgs)
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

func createSystemAdvisories(t *testing.T, systemID int, advisoryIDs []int,
	whenPatched *time.Time) {
	for _, advisoryID := range advisoryIDs {
		err := database.Db.Create(&models.SystemAdvisories{
			SystemID: systemID, AdvisoryID: advisoryID, WhenPatched: whenPatched}).Error
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

func deleteAdvisories(t *testing.T, advisories []string) {
	err := database.Db.Where("name IN (?)", advisories).
		Delete(&models.AdvisoryMetadata{}).Error
	assert.Nil(t, err)

	var advisoriesObjs []models.AdvisoryMetadata
	err = database.Db.Where("name IN (?)", advisories).
		Find(&advisoriesObjs).Error
	assert.Nil(t, err)
	assert.Equal(t, 0, len(advisoriesObjs))
}
