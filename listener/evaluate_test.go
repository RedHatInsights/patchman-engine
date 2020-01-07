package listener

import (
	"app/base/core"
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

var testDate = time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)

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
