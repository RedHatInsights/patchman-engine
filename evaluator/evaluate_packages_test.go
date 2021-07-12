package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"fmt"
	"testing"

	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/stretchr/testify/assert"
)

func TestAnalyzePackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()
	loadCache()

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
	updateList := map[string]vmaas.UpdatesV2ResponseUpdateList{}
	for i, name := range names {
		nevra := fmt.Sprintf("%s-%s", name, evras[i])
		updateList[nevra] = vmaas.UpdatesV2ResponseUpdateList{}
	}
	vmaasData := vmaas.UpdatesV2Response{UpdateList: &updateList}
	database.CheckEVRAsInDB(t, 0, evras...)
	err := lazySavePackages(database.Db, &vmaasData)
	assert.Nil(t, err)
	database.CheckEVRAsInDB(t, 2, evras[:2]...) // EVRAs were added
	database.CheckEVRAsInDB(t, 0, evras[2:]...) // EVRA for unknown package was not added
	database.DeleteNewlyAddedPackages(t)        // delete testing added package items
}
