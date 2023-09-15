package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVMaaSGetUpdates(t *testing.T) {
	utils.SkipWithoutPlatform(t)

	configure()
	vmaasData := getVMaaSUpdates(t)
	for k, v := range vmaasData.GetUpdateList() {
		if strings.HasPrefix(k, "firefox") {
			assert.Equal(t, 2, len(v.GetAvailableUpdates()))
		} else if strings.HasPrefix(k, "kernel") {
			assert.Equal(t, 1, len(v.GetAvailableUpdates()))
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
	aUpdates := []vmaas.UpdatesV3ResponseAvailableUpdates{
		{Erratum: utils.PtrString("ER1")}, {Erratum: utils.PtrString("ER2")}}
	bUpdates := []vmaas.UpdatesV3ResponseAvailableUpdates{
		{Erratum: utils.PtrString("ER2")}, {Erratum: utils.PtrString("ER3")}}
	cUpdates := []vmaas.UpdatesV3ResponseAvailableUpdates{
		{Erratum: utils.PtrString("ER3")}, {Erratum: utils.PtrString("ER4")}}
	updateList := map[string]*vmaas.UpdatesV3ResponseUpdateList{
		"pkg-a": {AvailableUpdates: &aUpdates},
		"pkg-b": {AvailableUpdates: &bUpdates},
		"pkg-c": {AvailableUpdates: &cUpdates},
	}
	vmaasData := vmaas.UpdatesV3Response{UpdateList: &updateList}
	advisories := getReportedAdvisories(&vmaasData)
	assert.Equal(t, 4, len(advisories))
}

func TestGetStoredAdvisoriesMap(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemAdvisories, err := getStoredAdvisoriesMap(database.Db, 1, 1)
	assert.Nil(t, err)
	assert.NotNil(t, systemAdvisories)
	assert.Equal(t, 8, len(systemAdvisories))
	assert.Equal(t, "RH-1", (systemAdvisories)["RH-1"].Advisory.Name)
	assert.Equal(t, "adv-1-des", (systemAdvisories)["RH-1"].Advisory.Description)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", (systemAdvisories)["RH-1"].Advisory.PublicDate.String())
}

func TestAdvisoryChanges(t *testing.T) {
	stored := database.CreateStoredAdvisories([]int64{1, 2, 3})
	reported := database.CreateReportedAdvisories([]string{"ER-1", "ER-3", "ER-4"},
		[]int{INSTALLABLE, INSTALLABLE, INSTALLABLE})
	patchedAIDs, installableNames, applicableNames := getAdvisoryChanges(reported, stored)
	assert.Equal(t, 1, len(installableNames))
	assert.Equal(t, "ER-4", installableNames[0])
	assert.Equal(t, 0, len(applicableNames))
	assert.Equal(t, 1, len(patchedAIDs))
	assert.Equal(t, int64(2), patchedAIDs[0])
}

func TestUpdatePatchedSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	system := models.SystemPlatform{ID: 12, RhAccountID: 3}
	advisoryIDs := []int64{2, 3, 4}
	database.CreateSystemAdvisories(t, system.RhAccountID, system.ID, advisoryIDs)
	database.CreateAdvisoryAccountData(t, system.RhAccountID, advisoryIDs, 1)
	// Update as-if the advisories had become patched
	err := updateAdvisoryAccountData(database.Db, &system, advisoryIDs, []int64{}, []int64{})
	assert.NoError(t, err)

	database.CheckSystemAdvisories(t, system.ID, advisoryIDs)
	database.CheckAdvisoriesAccountData(t, system.RhAccountID, advisoryIDs, 0)

	// Update as-if the advisories had become unpatched
	err = updateAdvisoryAccountData(database.Db, &system, []int64{}, advisoryIDs, []int64{})
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
	systemID := int64(2)
	advisoryIDs := []int64{2, 3, 4}
	err := ensureSystemAdvisories(database.Db, rhAccountID, systemID, advisoryIDs, []int64{})
	assert.Nil(t, err)
	database.CheckSystemAdvisories(t, systemID, advisoryIDs)
	database.DeleteSystemAdvisories(t, systemID, advisoryIDs)
}

func getVMaaSUpdates(t *testing.T) vmaas.UpdatesV3Response {
	ctx := context.Background()
	vmaasData := vmaas.UpdatesV3Response{}
	resp, err := vmaasClient.Request(&ctx, http.MethodPost, vmaasUpdatesURL, nil, &vmaasData)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Nil(t, resp.Body.Close())
	return vmaasData
}
