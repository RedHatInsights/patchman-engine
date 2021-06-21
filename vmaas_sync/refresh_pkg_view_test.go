package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRefreshAdvisoryCachesPerAccounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	// Create new latest kernel package
	desHash := []byte("11")
	sumHash := []byte("1")
	advisoryID := 7 // newer advisory ID
	evra := "5.6.14-200.fc31.x86_64"
	assert.NoError(t, database.Db.Create(&models.Package{
		NameID: 101, EVRA: evra, DescriptionHash: &desHash, SummaryHash: &sumHash,
		AdvisoryID: &advisoryID}).Error)
	refreshLatestPackagesView(0)

	var newPkgID int
	assert.NoError(t, database.Db.Table("package").Where("evra = ?", evra).
		Pluck("id", &newPkgID).Error)

	var latestKernelID int
	database.Db.Table("package_latest_cache").Where("name_id = 101").Pluck("package_id",  &latestKernelID)

	assert.Equal(t, newPkgID, latestKernelID)

	// database.DeleteNewlyAddedPackages(t)
	// refreshLatestPackagesView(0)
}
