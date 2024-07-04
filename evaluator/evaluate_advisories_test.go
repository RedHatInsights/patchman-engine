package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"context"
	"fmt"
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

func TestGetReportedAdvisoriesEmpty(t *testing.T) {
	utils.SkipWithoutPlatform(t)

	configure()
	available := vmaas.UpdatesV3ResponseAvailableUpdates{}
	update := vmaas.UpdatesV3ResponseUpdateList{
		AvailableUpdates: &[]vmaas.UpdatesV3ResponseAvailableUpdates{available},
	}
	updateList := map[string]*vmaas.UpdatesV3ResponseUpdateList{
		"package_without_erratum": &update,
	}
	vmaasData := vmaas.UpdatesV3Response{UpdateList: &updateList}
	advisories := getReportedAdvisories(&vmaasData)
	assert.Equal(t, 0, len(advisories))
}

func TestGetStoredAdvisoriesMap(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemAdvisories, err := loadSystemAdvisories(1, 1)
	assert.Nil(t, err)
	assert.NotNil(t, systemAdvisories)
	assert.Equal(t, 8, len(systemAdvisories))
	assert.Equal(t, "RH-1", (systemAdvisories)["RH-1"].Advisory.Name)
	assert.Equal(t, "adv-1-des", (systemAdvisories)["RH-1"].Advisory.Description)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", (systemAdvisories)["RH-1"].Advisory.PublicDate.String())
}

func TestAdvisoryChanges(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	stored := database.CreateStoredAdvisories([]int64{1, 2, 3})
	// create vmaasData with reported names: "ER-1", "ER-3", and "ER-4"
	updates := []vmaas.UpdatesV3ResponseAvailableUpdates{
		{Erratum: utils.PtrString("ER-1"), StatusID: INSTALLABLE},
		{Erratum: utils.PtrString("ER-3"), StatusID: APPLICABLE},
		{Erratum: utils.PtrString("ER-4"), StatusID: INSTALLABLE},
	}
	updateList := map[string]*vmaas.UpdatesV3ResponseUpdateList{
		"pkg-a": {AvailableUpdates: &updates},
	}
	vmaasData := vmaas.UpdatesV3Response{UpdateList: &updateList}

	// advisories must be lazy saved before evaluating changes
	err := lazySaveAdvisories(&vmaasData, inventoryID)
	defer database.DeleteNewlyAddedAdvisories(t)
	assert.Nil(t, err)

	extendedAdvisories, err := evaluateChanges(&vmaasData, stored)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(extendedAdvisories))

	assert.Equal(t, Keep, extendedAdvisories["ER-1"].Change)
	assert.Equal(t, Remove, extendedAdvisories["ER-2"].Change)
	assert.Equal(t, Update, extendedAdvisories["ER-3"].Change)
	assert.Equal(t, Add, extendedAdvisories["ER-4"].Change)
}

func TestUpdatePatchedSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	system := models.SystemPlatform{ID: 12, RhAccountID: 3}
	advisoryIDs := []int64{2, 3, 4}
	database.CreateSystemAdvisories(t, system.RhAccountID, system.ID, advisoryIDs)
	database.CreateAdvisoryAccountData(t, system.RhAccountID, advisoryIDs, 1)

	// Update as if the advisories became patched
	err := updateAdvisoryAccountData(database.DB, &system, advisoryIDs, SystemAdvisoryMap{})
	assert.NoError(t, err)
	database.CheckSystemAdvisories(t, system.ID, advisoryIDs)
	database.CheckAdvisoriesAccountData(t, system.RhAccountID, advisoryIDs, 0)

	// Update as if the advisories became unpatched
	systemAdvisories := make(SystemAdvisoryMap, len(advisoryIDs))
	for _, id := range advisoryIDs {
		systemAdvisories[fmt.Sprintf("ER-%v", id)] = models.SystemAdvisories{AdvisoryID: id}
	}
	err = updateAdvisoryAccountData(database.DB, &system, []int64{}, systemAdvisories)
	assert.NoError(t, err)
	database.CheckAdvisoriesAccountData(t, system.RhAccountID, advisoryIDs, 1)

	database.DeleteSystemAdvisories(t, system.ID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, system.RhAccountID, advisoryIDs)
}

func TestGetMissingAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	advisories := []string{"ER-1", "RH-1", "ER-2", "RH-2"}
	advisoryIDs := getAdvisoriesFromDB(advisories)
	missingNames, err := getMissingAdvisories(advisories)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(advisoryIDs))
	assert.Equal(t, 2, len(missingNames))
}

func TestGetMissingAdvisoriesEmptyString(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	advisories := []string{""}
	advisoryIDs := getAdvisoriesFromDB(advisories)
	missingNames, err := getMissingAdvisories(advisories)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(advisoryIDs))
	assert.Equal(t, 1, len(missingNames))
}

func TestProcessAndUpsertSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemID := int64(2)
	system := models.SystemPlatform{RhAccountID: 1, ID: systemID}
	extendedAdvisories := ExtendedAdvisoryMap{
		"ER-2": ExtendedAdvisory{Change: Keep, SystemAdvisories: models.SystemAdvisories{
			AdvisoryID: int64(2),
		}},
		"ER-3": ExtendedAdvisory{Change: Add, SystemAdvisories: models.SystemAdvisories{
			AdvisoryID: int64(3),
		}},
		"ER-4": ExtendedAdvisory{Change: Update, SystemAdvisories: models.SystemAdvisories{
			AdvisoryID: int64(4),
		}},
	}

	deleteIDs, advisoryObjs, updatedAdvisories := processAdvisories(&system, extendedAdvisories)
	assert.Equal(t, 0, len(deleteIDs))
	assert.Equal(t, 2, len(advisoryObjs))
	assert.Equal(t, len(updatedAdvisories), len(extendedAdvisories)-len(deleteIDs))

	err := upsertSystemAdvisories(database.DB, advisoryObjs)
	assert.Nil(t, err)
	database.CheckSystemAdvisories(t, systemID, []int64{3, 4})
	database.DeleteSystemAdvisories(t, systemID, []int64{3, 4})
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

func getAdvisoriesFromDB(advisories []string) []int64 {
	advisoryMetadata := make(models.AdvisoryMetadataSlice, 0, len(advisories))
	err := database.Db.Model(&models.AdvisoryMetadata{}).
		Where("name IN (?)", advisories).
		Select("id, name").
		Scan(&advisoryMetadata).Error
	if err != nil {
		return nil
	}

	advisoryIDs := make([]int64, 0, len(advisories))
	for _, am := range advisoryMetadata {
		advisoryIDs = append(advisoryIDs, am.ID)
	}
	return advisoryIDs
}
