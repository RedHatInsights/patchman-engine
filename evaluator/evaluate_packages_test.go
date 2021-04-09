package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

// New EVRAs for known package names will be added
func TestLazySavePackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	names := []string{"kernel", "firefox", "custom-package"}
	evras := []string{"1-0.el7.x86_64", "1-1.1.el7.x86_64", "11-1.el7.x86_64"}
	database.CheckEVRAsInDB(t, evras, 0)

	err := lazySavePackages(database.Db, names, evras)
	assert.Nil(t, err)
	database.CheckEVRAsInDB(t, evras[:2], 2)   // EVRAs were added
	database.CheckEVRAsInDB(t, evras[2:], 0)   // EVRA for unknown package was not added
	database.DeletePackagesWithEvras(t, evras) // delete testing added package items
}

func TestGetNameIDHashesMapCache(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	cache, err := getPackagesMetadata(database.Db)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(cache))
}
