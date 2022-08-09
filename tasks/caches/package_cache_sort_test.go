package caches

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

type CachedPackage struct {
	NameID    int
	PackageID int
	Summary   string
}

func TestPackageLatestCacheSort(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var packageLast CachedPackage
	database.Db.Table("package_latest_cache").Offset(1).First(&packageLast)
	assert.Equal(t, packageLast.NameID, 102)
	assert.Equal(t, packageLast.PackageID, 2)
}
