package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"fmt"
	"testing"

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
	// we send request with zero epoch and expect response with zero epoch
	// so we have to test with zero epoch
	vmaasData := vmaas.UpdatesV3Response{UpdateList: &map[string]vmaas.UpdatesV3ResponseUpdateList{
		"kernel-0:5.6.13-200.fc31.x86_64": {AvailableUpdates: &[]vmaas.UpdatesV3ResponseAvailableUpdates{}},
		"firefox-0:12.0.1-1.fc31.x86_64": {AvailableUpdates: &[]vmaas.UpdatesV3ResponseAvailableUpdates{{
			Package: utils.PtrString("firefox-0:77.0.1-1.fc31.x86_64"),
			EVRA:    utils.PtrString("0:77.0.1-1.fc31.x86_64"),
		}}},
		// this custom-package will NOT be ignored
		"custom-package-0:1.2.3-1.fc33.x86_64": {AvailableUpdates: &[]vmaas.UpdatesV3ResponseAvailableUpdates{{}}}}}

	installed, updatable, err := analyzePackages(database.Db, &system, &vmaasData)
	assert.Nil(t, err)
	assert.Equal(t, 3, installed)                                      // kernel, firefox, custom-package
	assert.Equal(t, 2, updatable)                                      // firefox, custom-package have updates
	database.CheckEVRAsInDBSynced(t, 1, false, "12.0.1-1.fc31.x86_64") // lazy added package
	database.CheckEVRAsInDB(t, 1, "1.2.3-1.fc33.x86_64")               // custom package is not ignored
	database.CheckSystemPackages(t, system.ID, 3)
	database.DeleteSystemPackages(t, system.ID)
	database.DeleteNewlyAddedPackages(t)
}

// New EVRAs for known package names will be added
func TestLazySavePackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()
	loadCache()

	names := []string{"kernel", "firefox", "custom-package"}
	evras := []string{"1-0.el7.x86_64", "1-1.1.el7.x86_64", "11-1.el7.x86_64"}
	updateList := make(map[string]vmaas.UpdatesV3ResponseUpdateList, len(names))
	for i, name := range names {
		nevra := fmt.Sprintf("%s-%s", name, evras[i])
		updateList[nevra] = vmaas.UpdatesV3ResponseUpdateList{}
	}
	vmaasData := vmaas.UpdatesV3Response{UpdateList: &updateList}
	database.CheckEVRAsInDB(t, 0, evras...)
	err := lazySavePackages(database.Db, &vmaasData)
	assert.Nil(t, err)
	database.CheckEVRAsInDBSynced(t, 2, false, evras[:2]...) // EVRAs were added
	database.CheckEVRAsInDB(t, 1, evras[2:]...)              // EVRA for unknown package was added
	database.DeleteNewlyAddedPackages(t)                     // delete testing added package items
}
