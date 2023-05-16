package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPkgListSyncPackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	Configure()

	var oldNameCount, oldPkgCount, newNameCount, newPkgCount int
	var pkgNextval, nameNextval, pkgCurrval, nameCurrval int

	database.Db.Model(&models.PackageName{}).Select("count(*)").Find(&oldNameCount)
	database.Db.Model(&models.Package{}).Select("count(*)").Find(&oldPkgCount)
	database.Db.Raw("select nextval('package_id_seq')").Find(&pkgNextval)
	database.Db.Raw("select nextval('package_name_id_seq')").Find(&nameNextval)

	err := syncPackages(time.Now(), nil)
	assert.NoError(t, err)

	// make sure we are not creating gaps in id sequences
	database.Db.Model(&models.PackageName{}).Select("count(*)").Find(&newNameCount)
	database.Db.Model(&models.Package{}).Select("count(*)").Find(&newPkgCount)
	database.Db.Raw("select currval('package_id_seq')").Find(&pkgCurrval)
	database.Db.Raw("select currval('package_name_id_seq')").Find(&nameCurrval)

	nameCountInc := newNameCount - oldNameCount
	nameMaxInc := nameCurrval - nameNextval
	pkgCountInc := newPkgCount - oldPkgCount
	pkgMaxInc := pkgCurrval - pkgNextval
	assert.Equal(t, nameCountInc, nameMaxInc)
	assert.Equal(t, pkgCountInc, pkgMaxInc)

	database.CheckPackagesNamesInDB(t, "", "bash", "curl")
	database.CheckPackagesNamesInDB(t, "summary like '% newest summary'", "bash", "curl")
	database.CheckEVRAsInDBSynced(t, 4, true,
		"0:77.0.1-1.fc31.src", "0:77.0.1-1.fc31.x86_64", // added firefox versions
		"0:5.7.13-200.fc31.src", "0:5.7.13-200.fc31.x86_64") // added kernel versions
	database.DeleteNewlyAddedPackages(t)
}
