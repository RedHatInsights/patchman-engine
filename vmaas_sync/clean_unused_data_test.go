package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanUnusedPackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	// package count before test
	var beforePkgCount int64
	err := database.Db.Model(models.Package{}).Count(&beforePkgCount).Error
	assert.Nil(t, err)

	// create unused package
	evra := "0000.0.0-0.x86_64"
	customPkg := models.Package{
		NameID: 101,
		EVRA:   evra,
		Synced: false,
	}
	err = database.Db.Create(&customPkg).Error
	assert.Nil(t, err)

	// package is there
	database.CheckEVRAsInDBSynced(t, 1, false, evra)

	// delete unused
	currentDeleteStatus := enableUnusedDataDelete
	enableUnusedDataDelete = true
	deleteUnusedPackages()
	enableUnusedDataDelete = currentDeleteStatus

	// is package deleted?
	database.CheckEVRAsInDB(t, 0, evra)

	// anything else deleted by mistake?
	var afterPkgCount int64
	err = database.Db.Model(models.Package{}).Count(&afterPkgCount).Error
	assert.Nil(t, err)
	assert.Equal(t, beforePkgCount, afterPkgCount)
}

// Test for making sure system culling works
func TestCleanUnusedAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	// advisory count before test
	var beforeAdvCount int64
	err := database.Db.Model(models.AdvisoryMetadata{}).Count(&beforeAdvCount).Error
	assert.Nil(t, err)

	// create unused advisory
	advisory := "CUSTOM-1234"
	customAdv := models.AdvisoryMetadata{
		Name:           advisory,
		Description:    "Custom desc",
		Synopsis:       "Custom syn",
		Summary:        "Custom sum",
		Solution:       utils.PtrString("Custom sol"),
		AdvisoryTypeID: 1,
		RebootRequired: false,
		Synced:         false,
	}
	err = database.Db.Create(&customAdv).Error
	assert.Nil(t, err)

	// advisory is there
	database.CheckAdvisoriesInDB(t, []string{advisory})

	// delete unused
	currentDeleteStatus := enableUnusedDataDelete
	enableUnusedDataDelete = true
	deleteUnusedAdvisories()
	enableUnusedDataDelete = currentDeleteStatus

	// is custom advisory deleted?
	var count int64
	err = database.Db.Model(models.AdvisoryMetadata{}).Where("name = ?", advisory).
		Count(&count).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(0), count)

	// anything else deleted by mistake?
	var afterAdvCount int64
	err = database.Db.Model(models.AdvisoryMetadata{}).Count(&afterAdvCount).Error
	assert.Nil(t, err)
	assert.Equal(t, beforeAdvCount, afterAdvCount)
}
