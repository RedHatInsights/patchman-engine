package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPackageCache(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	pc := NewPackageCache(true, true, 11, 11)
	assert.NotNil(t, pc)
	pc.Load()
	assert.Equal(t, 11, pc.byID.Len())
	assert.Equal(t, 11, pc.byNevra.Len())
	assert.Equal(t, 7, pc.latestByName.Len())
	assert.Equal(t, 7, pc.nameByID.Len())
	// ask for a package not in cache
	val, ok := pc.GetByID(1)
	assert.True(t, ok)
	assert.Equal(t, "kernel", val.Name)
	assert.Equal(t, "5.6.13-200.fc31.x86_64", val.Evra)
	assert.Equal(t, int64(101), val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
	// ask for a package already in cache
	val, ok = pc.GetByNevra("kernel-0:5.6.13-201.fc31.x86_64")
	pkgIDs := database.GetPackageIDs("kernel-0:5.6.13-201.fc31.x86_64", "kernel-0:5.10.13-200.fc31.x86_64")
	assert.True(t, ok)
	assert.Equal(t, pkgIDs[0], val.ID)
	assert.Equal(t, "kernel", val.Name)
	assert.Equal(t, "5.6.13-201.fc31.x86_64", val.Evra)
	assert.Equal(t, int64(101), val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
	val, ok = pc.GetLatestByName("kernel")
	assert.True(t, ok)
	assert.Equal(t, pkgIDs[1], val.ID)
	assert.Equal(t, "kernel", val.Name)
	assert.Equal(t, "5.10.13-200.fc31.x86_64", val.Evra)
	assert.Equal(t, int64(101), val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
	valStr, ok := pc.GetNameByID(104)
	assert.True(t, ok)
	assert.Equal(t, "curl", valStr)
}

func TestGetByNevras(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	pc := NewPackageCache(true, true, 11, 11)
	assert.NotNil(t, pc)
	pc.Load()
	// ask for a package not in cache
	val, missingDB, ok := pc.GetByNevras([]string{"kernel-0:5.6.13-201.fc31.x86_64"})
	assert.True(t, ok)
	assert.Len(t, val, 1)
	assert.Len(t, missingDB, 0)
	assert.Equal(t, "kernel", val[0].Name)
	assert.Equal(t, "5.6.13-201.fc31.x86_64", val[0].Evra)

	// ask for a package in cache and a new package not in DB
	val, missingDB, ok = pc.GetByNevras([]string{
		"kernel-0:5.6.13-201.fc31.x86_64",
		"get-by-nevras-test-0:1.1.1-1.el9.x86_64",
		"invalidnevra-1.1",
	})
	assert.True(t, ok)
	assert.Len(t, val, 1)
	assert.Equal(t, "kernel", val[0].Name)
	assert.Equal(t, "5.6.13-201.fc31.x86_64", val[0].Evra)
	assert.Len(t, missingDB, 1)
	for nevra, pkg := range missingDB {
		assert.Equal(t, "get-by-nevras-test-0:1.1.1-1.el9.x86_64", nevra)
		assert.Equal(t, "get-by-nevras-test", pkg.Name)
		assert.Equal(t, 0, pkg.Epoch)
		assert.Equal(t, "1.1.1", pkg.Version)
		assert.Equal(t, "1.el9", pkg.Release)
		assert.Equal(t, "x86_64", pkg.Arch)
	}

	// ask for a package not in cache and not in db
	val, missingDB, ok = pc.GetByNevras([]string{"get-by-nevras-test-0:1.1.1-1.el9.x86_64"})
	assert.False(t, ok)
	assert.Empty(t, val)
	assert.Len(t, missingDB, 1)
}

func TestMissPackageCache(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	pc := NewPackageCache(true, true, 11, 11)
	assert.NotNil(t, pc)
	pc.Load()
	// try non existent packages
	val, ok := pc.GetByID(9999)
	assert.Nil(t, val)
	assert.False(t, ok)
	val, ok = pc.GetByNevra("kernel-0:1.2.13-1.i386")
	assert.Nil(t, val)
	assert.False(t, ok)
	val, ok = pc.GetLatestByName("dummy")
	assert.Nil(t, val)
	assert.False(t, ok)
	valStr, ok := pc.GetNameByID(9999)
	assert.Empty(t, valStr)
	assert.False(t, ok)
}

func TestAddPackageCache(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	pc := NewPackageCache(true, true, 100, 100)
	assert.NotNil(t, pc)
	pc.Load()
	assert.Equal(t, 17, pc.byID.Len())
	assert.Equal(t, 17, pc.byNevra.Len())
	assert.Equal(t, 11, pc.latestByName.Len())
	assert.Equal(t, 11, pc.nameByID.Len())

	pkg := PackageCacheMetadata{
		ID:              999,
		NameID:          104,
		Name:            "curl",
		Evra:            "999-999.x86_64",
		DescriptionHash: []byte("44"),
		SummaryHash:     []byte("4"),
	}
	pc.Add(&pkg)
	assert.Equal(t, 18, pc.byID.Len())
	assert.Equal(t, 18, pc.byNevra.Len())
	assert.Equal(t, 11, pc.latestByName.Len())
	assert.Equal(t, 11, pc.nameByID.Len())

	val, ok := pc.GetByID(999)
	assert.True(t, ok)
	assert.Equal(t, pkg.Name, val.Name)
	assert.Equal(t, pkg.Evra, val.Evra)
	assert.Equal(t, pkg.NameID, val.NameID)
	assert.Equal(t, pkg.DescriptionHash, val.DescriptionHash)
	assert.Equal(t, pkg.SummaryHash, val.SummaryHash)
}
