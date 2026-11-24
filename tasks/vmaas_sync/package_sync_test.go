package vmaas_sync

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"crypto/sha512"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStringPtr2HashSHA512(t *testing.T) {
	value := "package summary"
	got := stringPtr2Hash(&value)
	sum := sha512.Sum512([]byte(value))
	assert.Equal(t, sum[:], *got)

	assert.Nil(t, stringPtr2Hash(nil))
	empty := ""
	assert.Nil(t, stringPtr2Hash(&empty))
}

func TestPkgListSyncPackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	Configure()

	var oldNameCount, oldPkgCount, newNameCount, newPkgCount int
	var pkgNextval, nameNextval, pkgCurrval, nameCurrval int

	database.TestDB().Model(&models.PackageName{}).Select("count(*)").Find(&oldNameCount)
	database.TestDB().Model(&models.Package{}).Select("count(*)").Find(&oldPkgCount)
	database.TestDB().Raw("select nextval('package_id_seq')").Find(&pkgNextval)
	database.TestDB().Raw("select nextval('package_name_id_seq')").Find(&nameNextval)

	err := syncPackages(time.Now(), nil)
	assert.NoError(t, err)

	// make sure we are not creating gaps in id sequences
	database.TestDB().Model(&models.PackageName{}).Select("count(*)").Find(&newNameCount)
	database.TestDB().Model(&models.Package{}).Select("count(*)").Find(&newPkgCount)
	database.TestDB().Raw("select currval('package_id_seq')").Find(&pkgCurrval)
	database.TestDB().Raw("select currval('package_name_id_seq')").Find(&nameCurrval)

	nameCountInc := newNameCount - oldNameCount
	nameMaxInc := nameCurrval - nameNextval
	pkgCountInc := newPkgCount - oldPkgCount
	pkgMaxInc := pkgCurrval - pkgNextval
	assert.Equal(t, nameCountInc, nameMaxInc)
	assert.Equal(t, pkgCountInc, pkgMaxInc)

	database.CheckPackagesNamesInDB(t, "", "bash", "curl")
	database.CheckPackagesNamesInDB(t, "summary like '% newest summary'", "bash", "curl")
	database.CheckEVRAsInDBSynced(t, 4, true,
		"77.0.1-1.fc31.src", "77.0.1-1.fc31.x86_64", // added firefox versions
		"5.7.13-200.fc31.src", "5.7.13-200.fc31.x86_64") // added kernel versions
	database.DeleteNewlyAddedPackages(t)
}
