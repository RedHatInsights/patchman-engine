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
	"strings"
	"testing"
	"time"
)

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
	advisories := getReportedAdvisories(&vmaasData)
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
	advisories := getReportedAdvisories(&vmaasData)
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
	stored := database.CreateStoredAdvisories(map[int]*time.Time{1: &testDate, 2: nil, 3: nil})
	reported := database.CreateReportedAdvisories("ER-1", "ER-3", "ER-4")
	news, unpatched := getNewAndUnpatchedAdvisories(reported, stored)
	assert.Equal(t, 1, len(news))
	assert.Equal(t, "ER-4", news[0])
	assert.Equal(t, 1, len(unpatched))
	assert.Equal(t, 1, unpatched[0])
}

func TestGetPatchedAdvisories(t *testing.T) {
	stored := database.CreateStoredAdvisories(map[int]*time.Time{1: &testDate, 2: nil, 3: nil})
	reported := database.CreateReportedAdvisories("ER-3", "ER-4")
	patched := getPatchedAdvisories(reported, stored)
	assert.Equal(t, 1, len(patched))
	assert.Equal(t, 2, patched[0])
}

func TestUpdatePatchedSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	system := models.SystemPlatform{ID: 12, RhAccountID: 3}
	advisoryIDs := []int{2, 3, 4}
	database.CreateSystemAdvisories(t, system.RhAccountID, system.ID, advisoryIDs, nil)
	database.CreateAdvisoryAccountData(t, system.RhAccountID, advisoryIDs, 1)
	database.UpdateSystemAdvisoriesWhenPatched(t, system.ID, system.RhAccountID, advisoryIDs, &testDate)
	// Update as-if the advisories had become patched
	err := updateAdvisoryAccountDatas(database.Db, &system, advisoryIDs, []int{})
	assert.NoError(t, err)

	database.CheckSystemAdvisoriesWhenPatched(t, system.ID, advisoryIDs, &testDate)
	database.CheckAdvisoriesAccountData(t, system.RhAccountID, advisoryIDs, 0)
	database.UpdateSystemAdvisoriesWhenPatched(t, system.ID, system.RhAccountID, advisoryIDs, nil)

	// Update as-if the advisories had become unpatched
	err = updateAdvisoryAccountDatas(database.Db, &system, []int{}, advisoryIDs)
	assert.NoError(t, err)

	database.CheckAdvisoriesAccountData(t, system.RhAccountID, advisoryIDs, 1)
	database.DeleteSystemAdvisories(t, system.ID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, system.RhAccountID, advisoryIDs)
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
	database.CheckSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, nil)
	database.DeleteSystemAdvisories(t, systemID, advisoryIDs)
}

func getVMaaSUpdates(t *testing.T) vmaas.UpdatesV2Response {
	vmaasCallArgs := vmaas.AppUpdatesHandlerV3PostPostOpts{}
	vmaasData, resp, err := vmaasClient.DefaultApi.AppUpdatesHandlerV3PostPost(context.Background(), &vmaasCallArgs)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Nil(t, resp.Body.Close())
	return vmaasData
}
