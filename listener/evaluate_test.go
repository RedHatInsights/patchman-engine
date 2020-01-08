package listener

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

	configure()
	vmaasData := getVMaaSUpdates(t)
	assert.Equal(t, 2, len(vmaasData.UpdateList["firefox"].AvailableUpdates))
	assert.Equal(t, 1, len(vmaasData.UpdateList["kernel"].AvailableUpdates))
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

	systemAdvisories, err := getStoredAdvisoriesMap(0)
	assert.Nil(t, err)
	assert.NotNil(t, systemAdvisories)
	assert.Equal(t, 8, len(*systemAdvisories))
	assert.Equal(t, "RH-1", (*systemAdvisories)["RH-1"].Advisory.Name)
	assert.Equal(t, "adv-1-des", (*systemAdvisories)["RH-1"].Advisory.Description)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", (*systemAdvisories)["RH-1"].Advisory.PublicDate.String())
}

func TestGetNewAndUnpatchedAdvisories(t *testing.T) {
	stored := createTestStoredAdvisories(map[int]*time.Time{1: &testDate, 2: nil, 3: nil})
	reported := createTestReportedAdvisories("ER-1", "ER-3", "ER-4")
	news, unpatched := getNewAndUnpatchedAdvisories(reported, stored)
	assert.Equal(t, 1, len(news))
	assert.Equal(t, "ER-4", news[0])
	assert.Equal(t, 1, len(unpatched))
	assert.Equal(t, 1, unpatched[0])
}

func TestGetPatchedAdvisories(t *testing.T) {
	stored := createTestStoredAdvisories(map[int]*time.Time{1: &testDate, 2: nil, 3: nil})
	reported := createTestReportedAdvisories("ER-3", "ER-4")
	patched := getPatchedAdvisories(reported, stored)
	assert.Equal(t, 1, len(patched))
	assert.Equal(t, 2, patched[0])
}

func TestUpdatePatchedSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemID := 1
	advisoryIDs := []int{2, 3, 4}
	createTestingSystemAdvisories(t, systemID, advisoryIDs, nil)

	err := updateSystemAdvisoriesWhenPatched(systemID, advisoryIDs, &testDate)
	assert.Nil(t, err)
	checkSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, &testDate)

	deleteTestingSystemAdvisories(t, systemID, advisoryIDs)
}

func TestUpdateUnpatchedSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemID := 1
	advisoryIDs := []int{2, 3, 4}
	createTestingSystemAdvisories(t, systemID, advisoryIDs, &testDate)

	err := updateSystemAdvisoriesWhenPatched(systemID, advisoryIDs, nil)
	assert.Nil(t, err)
	checkSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, nil)

	deleteTestingSystemAdvisories(t, systemID, advisoryIDs)
}

func TestEvaluate(t *testing.T) {
	utils.SkipWithoutPlatform(t)
	utils.SkipWithoutDB(t)

	configure()
	evaluate(12, 2, context.Background(), vmaas.UpdatesRequest{})
}

func getVMaaSUpdates(t *testing.T) vmaas.UpdatesV2Response {
	vmaasCallArgs := vmaas.AppUpdatesHandlerV2PostPostOpts{}
	vmaasData, resp, err := vmaasClient.UpdatesApi.AppUpdatesHandlerV2PostPost(context.Background(), &vmaasCallArgs)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	return vmaasData
}

func createTestReportedAdvisories(reportedAdvisories ...string) map[string]bool {
	reportedAdvisoriesMap := map[string]bool{}
	for _, adv := range reportedAdvisories {
		reportedAdvisoriesMap[adv] = true
	}
	return reportedAdvisoriesMap
}

func createTestStoredAdvisories(advisoryPatched map[int]*time.Time) map[string]models.SystemAdvisories {
	systemAdvisoriesMap := map[string]models.SystemAdvisories{}
	for advisoryID, patched := range advisoryPatched {
		systemAdvisoriesMap["ER-" + strconv.Itoa(advisoryID)] = models.SystemAdvisories{
			WhenPatched: patched,
			AdvisoryID: advisoryID}
	}
	return systemAdvisoriesMap
}

func createTestingSystemAdvisories(t *testing.T, systemID int, advisoryIDs []int, whenPatched *time.Time) {
	for i, advisoryID := range advisoryIDs {
		err := database.Db.Create(&models.SystemAdvisories{
			ID: (i + 1) * 100, SystemID: systemID, AdvisoryID: advisoryID, WhenPatched: whenPatched}).Error
		assert.Nil(t, err)
	}
	checkSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, whenPatched)
}

func checkSystemAdvisoriesWhenPatched(t *testing.T, systemID int, advisoryIDs []int, whenPatched *time.Time) {
	var systemAdvisories []models.SystemAdvisories
	err := database.Db.Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
		Find(&systemAdvisories).Error
	assert.Nil(t, err)
	assert.Equal(t, len(advisoryIDs), len(systemAdvisories))
	for _, systemAdvisory := range systemAdvisories {
		if whenPatched == nil {
			assert.Nil(t, systemAdvisory.WhenPatched)
		} else {
			assert.Equal(t, systemAdvisory.WhenPatched.String(), whenPatched.String())
		}
	}
}

func deleteTestingSystemAdvisories(t *testing.T, systemID int, advisoryIDs []int) {
	err := database.Db.Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
		Delete(&models.SystemAdvisories{}).Error
	assert.Nil(t, err)

	var systemAdvisories []models.SystemAdvisories
	err = database.Db.Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
		Find(&systemAdvisories).Error
	assert.Nil(t, err)
	assert.Equal(t, 0, len(systemAdvisories))
}
