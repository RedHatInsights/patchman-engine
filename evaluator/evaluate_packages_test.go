package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAnalyzePackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	system := models.SystemPlatform{ID: 11, RhAccountID: 2}
	database.CheckSystemPackages(t, system.ID, 0)
	database.CheckEVRAsInDB(t, 0, "12.0.1-1.fc31.x86_64") // lazy added package
	vmaasData := vmaas.UpdatesV2Response{UpdateList: &map[string]vmaas.UpdatesV2ResponseUpdateList{
		"kernel-5.6.13-200.fc31.x86_64": {AvailableUpdates: &[]vmaas.UpdatesV2ResponseAvailableUpdates{}},
		"firefox-12.0.1-1.fc31.x86_64": {AvailableUpdates: &[]vmaas.UpdatesV2ResponseAvailableUpdates{{
			Package: utils.PtrString("firefox-77.0.1-1.fc31.x86_64"),
		}}},
		// this custom-package will be ignored
		"custom-package-1.2.3-1.fc33.x86_64": {AvailableUpdates: &[]vmaas.UpdatesV2ResponseAvailableUpdates{{}}}}}

	installed, updatable, err := analyzePackages(database.Db, &system, &vmaasData)
	assert.Nil(t, err)
	assert.Equal(t, 2, installed)                         // kernel, firefox
	assert.Equal(t, 1, updatable)                         // firefox has updates
	database.CheckEVRAsInDB(t, 1, "12.0.1-1.fc31.x86_64") // lazy added package
	database.CheckEVRAsInDB(t, 0, "1.2.3-1.fc33.x86_64")  // ignored custom package
	database.CheckSystemPackages(t, system.ID, 2)
	database.DeleteSystemPackages(t, system.ID)
	database.DeleteNewlyAddedPackages(t)
}

// New EVRAs for known package names will be added
func TestLazySavePackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	names := []string{"kernel", "firefox", "custom-package"}
	evras := []string{"1-0.el7.x86_64", "1-1.1.el7.x86_64", "11-1.el7.x86_64"}
	database.CheckEVRAsInDB(t, 0, evras...)

	err := lazySavePackages(database.Db, names, evras)
	assert.Nil(t, err)
	database.CheckEVRAsInDB(t, 2, evras[:2]...) // EVRAs were added
	database.CheckEVRAsInDB(t, 0, evras[2:]...) // EVRA for unknown package was not added
	database.DeleteNewlyAddedPackages(t)        // delete testing added package items
}

func TestGetNameIDHashesMapCache(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	cache, err := getPackagesMetadata(database.Db)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(cache))
	val, ok := cache["kernel"]
	assert.True(t, ok)
	assert.Equal(t, 101, val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
}
