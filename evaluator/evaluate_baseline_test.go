package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"

	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitVmaasToBaseline(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	// a system without baseline
	system := models.SystemPlatform{ID: 5, RhAccountID: 1, BaselineID: nil}
	originalVmaasData := getVMaaSUpdates(t)
	vmaasData := getVMaaSUpdates(t)
	err := limitVmaasToBaseline(database.Db, &system, &vmaasData)
	assert.Nil(t, err)
	assert.Equal(t, originalVmaasData, vmaasData)

	// a system with baseline but nothing filtered out
	system = models.SystemPlatform{ID: 3, RhAccountID: 1, BaselineID: utils.PtrInt64(2)}
	err = limitVmaasToBaseline(database.Db, &system, &vmaasData)
	assert.Nil(t, err)
	assert.Equal(t, []string{"RH-1", "RH-100", "RH-2"}, errataInVmaasData(vmaasData, INSTALLABLE))

	// a system with baseline and filtered errata
	system = models.SystemPlatform{ID: 1, RhAccountID: 1, BaselineID: utils.PtrInt64(1)}
	vmaasData = getVMaaSUpdates(t)
	err = limitVmaasToBaseline(database.Db, &system, &vmaasData)
	assert.Nil(t, err)
	assert.Equal(t, []string{"RH-100"}, errataInVmaasData(vmaasData, INSTALLABLE))
	assert.Equal(t, []string{"RH-1", "RH-2"}, errataInVmaasData(vmaasData, APPLICABLE))
}

func errataInVmaasData(vmaasData vmaas.UpdatesV2Response, status int) []string {
	errata := make([]string, 0)
	for _, updates := range vmaasData.GetUpdateList() {
		availableUpdates := updates.GetAvailableUpdates()
		for _, u := range availableUpdates {
			if u.StatusID == status {
				advisoryName := u.GetErratum()
				errata = append(errata, advisoryName)
			}
		}
	}
	sort.Strings(errata)
	return errata
}
