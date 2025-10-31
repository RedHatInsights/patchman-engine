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
	vmaasData := mockVMaaSResponse()
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

func TestLoadSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemAdvisories, err := loadSystemAdvisories(database.DB, 1, 1)
	assert.Nil(t, err)
	assert.NotNil(t, systemAdvisories)
	assert.Equal(t, 8, len(systemAdvisories))
	assert.Equal(t, "RH-1", (systemAdvisories)["RH-1"].Advisory.Name)
	assert.Equal(t, "adv-1-des", (systemAdvisories)["RH-1"].Advisory.Description)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", (systemAdvisories)["RH-1"].Advisory.PublicDate.String())
}

func TestEvaluateChanges(t *testing.T) {
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

	assert.Equal(t, Keep, extendedAdvisories["ER-1"].change)
	assert.Equal(t, Remove, extendedAdvisories["ER-2"].change)
	assert.Equal(t, Update, extendedAdvisories["ER-3"].change)
	assert.Equal(t, Add, extendedAdvisories["ER-4"].change)
}

func TestLoadMissingNamesIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	vmaasData := mockVMaaSResponse()
	missingNames := []string{"ER1", "ER2", "ER3", "ER4"}
	extendedAdvisories := extendedAdvisoryMap{"ER1": {}, "ER2": {}, "ER3": {}, "ER4": {}}

	// test error if not lazy saved
	err := loadMissingNamesIDs(missingNames, extendedAdvisories)
	assert.Error(t, err)

	// test OK if lazy saved
	err = lazySaveAdvisories(&vmaasData, inventoryID)
	defer database.DeleteNewlyAddedAdvisories(t)
	assert.NoError(t, err)
	err = loadMissingNamesIDs(missingNames, extendedAdvisories)
	assert.NoError(t, err)
	assert.NotEqual(t, int64(0), extendedAdvisories["ER1"].AdvisoryID)
	assert.NotEqual(t, int64(0), extendedAdvisories["ER2"].AdvisoryID)
	assert.NotEqual(t, int64(0), extendedAdvisories["ER3"].AdvisoryID)
	assert.NotEqual(t, int64(0), extendedAdvisories["ER4"].AdvisoryID)
}

func TestIncrementAdvisoryTypeCounts(t *testing.T) {
	var (
		enhCount int
		bugCount int
		secCount int
	)
	advisories := []models.AdvisoryMetadata{
		{AdvisoryTypeID: enhancement},
		{AdvisoryTypeID: bugfix},
		{AdvisoryTypeID: security},
	}

	for _, advisory := range advisories {
		incrementAdvisoryTypeCounts(advisory, &enhCount, &bugCount, &secCount)
	}
	assert.Equal(t, 1, enhCount)
	assert.Equal(t, 1, bugCount)
	assert.Equal(t, 1, secCount)
}

func TestUpdateAdvisoryAccountData(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	system := models.SystemPlatform{ID: 12, RhAccountID: 3}
	advisoryIDs := []int64{2, 3, 4}
	database.CreateSystemAdvisories(t, system.RhAccountID, system.ID, advisoryIDs)
	database.CreateAdvisoryAccountData(t, system.RhAccountID, advisoryIDs, 1)
	advisoriesByName := extendedAdvisoryMap{
		"ER-2": {
			change:           Remove,
			SystemAdvisories: models.SystemAdvisories{AdvisoryID: 2, SystemID: system.ID, RhAccountID: system.RhAccountID},
		},
		"ER-3": {
			change:           Remove,
			SystemAdvisories: models.SystemAdvisories{AdvisoryID: 3, SystemID: system.ID, RhAccountID: system.RhAccountID},
		},
		"ER-4": {
			change:           Remove,
			SystemAdvisories: models.SystemAdvisories{AdvisoryID: 4, SystemID: system.ID, RhAccountID: system.RhAccountID},
		},
	}

	// Update as if the advisories became patched
	err := updateAdvisoryAccountData(database.DB, &system, advisoriesByName)
	assert.NoError(t, err)
	database.CheckSystemAdvisories(t, system.ID, advisoryIDs)
	database.CheckAdvisoriesAccountData(t, system.RhAccountID, advisoryIDs, 0)

	// Update as if the advisories became unpatched
	for name, ea := range advisoriesByName {
		ea.change = Add
		advisoriesByName[name] = ea
	}
	err = updateAdvisoryAccountData(database.DB, &system, advisoriesByName)
	assert.NoError(t, err)
	database.CheckAdvisoriesAccountData(t, system.RhAccountID, advisoryIDs, 1)

	database.DeleteSystemAdvisories(t, system.ID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, system.RhAccountID, advisoryIDs)
}

func TestGetMissingAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	advisoryNames := []string{"ER-1", "RH-1", "ER-2", "RH-2"}
	advisoryMap := map[string]int{"ER-1": 0, "RH-1": 0, "ER-2": 0, "RH-2": 0}
	advisoryIDs := getAdvisoryIDsByNames(t, advisoryNames)
	missingNames, err := getMissingAdvisories(advisoryMap)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(advisoryIDs))
	assert.Equal(t, 2, len(missingNames))
}

func TestGetMissingAdvisoriesEmptyString(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	advisories := []string{""}
	advisoryIDs := getAdvisoryIDsByNames(t, advisories)
	missingNames, err := getMissingAdvisories(map[string]int{"": 0})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(advisoryIDs))
	assert.Equal(t, 1, len(missingNames))
}

func TestProcessAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	systemID := int64(2)
	system := models.SystemPlatform{RhAccountID: 1, ID: systemID}
	extendedAdvisories := extendedAdvisoryMap{
		"ER-2": extendedAdvisory{change: Keep, SystemAdvisories: models.SystemAdvisories{
			AdvisoryID: int64(2),
		}},
		"ER-3": extendedAdvisory{change: Add, SystemAdvisories: models.SystemAdvisories{
			AdvisoryID: int64(3),
		}},
		"ER-4": extendedAdvisory{change: Update, SystemAdvisories: models.SystemAdvisories{
			AdvisoryID: int64(4),
		}},
	}

	deleteIDs, advisoryObjs := processAdvisories(&system, extendedAdvisories)
	assert.Equal(t, 0, len(deleteIDs))
	assert.Equal(t, 2, len(advisoryObjs))
}

func TestUpsertSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	// ensure consistent environment
	database.DeleteSystemAdvisories(t, systemID, []int64{3, 4})
	database.CreateSystemAdvisories(t, rhAccountID, systemID, []int64{3})

	// mock data: the system advisory with ID=3 exists and will be updated,
	// the system advisory with ID=4 will be created
	advisoryObjs := models.SystemAdvisoriesSlice{
		models.SystemAdvisories{SystemID: systemID, RhAccountID: rhAccountID,
			AdvisoryID: int64(3),
			StatusID:   APPLICABLE,
		},
		models.SystemAdvisories{SystemID: systemID, RhAccountID: rhAccountID,
			AdvisoryID: int64(4),
		},
	}

	// check insert
	err := upsertSystemAdvisories(database.DB, advisoryObjs)
	assert.Nil(t, err)
	database.CheckSystemAdvisories(t, systemID, []int64{3, 4})

	// check update
	var updatedAdvisory models.SystemAdvisories
	err = database.DB.Model(models.SystemAdvisories{}).Find(&updatedAdvisory, []int64{3}).Error
	assert.Nil(t, err)
	assert.Equal(t, APPLICABLE, updatedAdvisory.StatusID)

	// cleanup
	database.DeleteSystemAdvisories(t, systemID, []int64{3, 4})
}

func TestCalcAdvisoryChanges(t *testing.T) {
	system := models.SystemPlatform{ID: systemID, RhAccountID: rhAccountID}
	advisoriesByName := extendedAdvisoryMap{
		"ER-102": {
			change:           Update,
			SystemAdvisories: models.SystemAdvisories{AdvisoryID: int64(102), StatusID: INSTALLABLE},
		},
		"ER-103": {
			change:           Remove,
			SystemAdvisories: models.SystemAdvisories{AdvisoryID: int64(103), StatusID: INSTALLABLE},
		},
		"ER-104": {
			change:           Remove,
			SystemAdvisories: models.SystemAdvisories{AdvisoryID: int64(104), StatusID: APPLICABLE},
		},
		"ER-105": {
			change:           Add,
			SystemAdvisories: models.SystemAdvisories{AdvisoryID: int64(105), StatusID: APPLICABLE},
		},
	}

	changes := calcAdvisoryChanges(&system, advisoriesByName)
	expected := map[int64]models.AdvisoryAccountData{
		102: {SystemsApplicable: 1, SystemsInstallable: 1},
		103: {SystemsApplicable: -1, SystemsInstallable: -1},
		104: {SystemsInstallable: -1},
		105: {SystemsApplicable: 1},
	}
	assert.Equal(t, len(expected), len(changes))
	for _, change := range changes {
		advisoryID := change.AdvisoryID
		assert.Equal(t, change.SystemsApplicable, expected[advisoryID].SystemsApplicable)
		assert.Equal(t, change.SystemsInstallable, expected[advisoryID].SystemsInstallable)
	}
}

func TestStoreMissingAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	otherAdvisories := []string{"ER-TSMA-1", "ER-TSMA-2", "ER-TSMA-3", "ER-TSMA-4"}
	err := storeMissingAdvisories(otherAdvisories)
	assert.NoError(t, err)
	advisories := database.GetAdvisoriesByName(t, otherAdvisories)
	expectedText := "Not Available for 3rd party systems"
	for _, advisory := range advisories {
		assert.Equal(t, expectedText, advisory.Description)
		assert.Equal(t, expectedText, advisory.Synopsis)
		assert.Equal(t, expectedText, advisory.Summary)
	}
	database.DeleteAdvisoriesByName(t, otherAdvisories)

	rhelAdvisories := []string{"RHSA-TSMA-1", "RHEA-TSMA-2", "RHBA-TSMA-3"}
	err = storeMissingAdvisories(rhelAdvisories)
	assert.NoError(t, err)
	advisories = database.GetAdvisoriesByName(t, rhelAdvisories)
	for _, advisory := range advisories {
		expectedText = fmt.Sprintf("https://access.redhat.com/errata/%s", advisory.Name)
		assert.Equal(t, expectedText, advisory.Description)
		assert.Equal(t, expectedText, advisory.Synopsis)
		assert.Equal(t, expectedText, advisory.Summary)
	}
	database.DeleteAdvisoriesByName(t, rhelAdvisories)
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

func mockVMaaSResponse() vmaas.UpdatesV3Response {
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
	return vmaas.UpdatesV3Response{UpdateList: &updateList}
}

func getAdvisoryIDsByNames(t *testing.T, names []string) []int64 {
	ids := make([]int64, 0, len(names))
	err := database.DB.Model(&models.AdvisoryMetadata{}).
		Where("name IN (?)", names).
		Pluck("id", &ids).Error
	assert.NoError(t, err)
	return ids
}
