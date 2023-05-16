package evaluator

import (
	"app/base/core"
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
	assert.Equal(t, 11, pc.latestByName.Len())
	assert.Equal(t, 11, pc.nameByID.Len())
	// ask for a package not in cache
	val, ok := pc.GetByID(1)
	assert.True(t, ok)
	assert.Equal(t, "kernel", val.Name)
	assert.Equal(t, "0:5.6.13-200.fc31.x86_64", val.Evra)
	assert.Equal(t, int64(101), val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
	// ask for a package already in cache
	val, ok = pc.GetByNevra("kernel-0:5.6.13-201.fc31.x86_64")
	assert.True(t, ok)
	assert.Equal(t, int64(11), val.ID)
	assert.Equal(t, "kernel", val.Name)
	assert.Equal(t, "0:5.6.13-201.fc31.x86_64", val.Evra)
	assert.Equal(t, int64(101), val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
	val, ok = pc.GetLatestByName("kernel")
	assert.True(t, ok)
	assert.Equal(t, int64(11), val.ID)
	assert.Equal(t, "kernel", val.Name)
	assert.Equal(t, "0:5.6.13-201.fc31.x86_64", val.Evra)
	assert.Equal(t, int64(101), val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
	valStr, ok := pc.GetNameByID(104)
	assert.True(t, ok)
	assert.Equal(t, "curl", valStr)
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
	assert.Equal(t, 13, pc.byID.Len())
	assert.Equal(t, 13, pc.byNevra.Len())
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
	assert.Equal(t, 14, pc.byID.Len())
	assert.Equal(t, 14, pc.byNevra.Len())
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
