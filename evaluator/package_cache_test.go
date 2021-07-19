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

	pc := NewPackageCache()
	assert.NotNil(t, pc)
	pc.Load()
	assert.Equal(t, 12, len(pc.byID))
	assert.Equal(t, 12, len(pc.byNevra))
	assert.Equal(t, 10, len(pc.latestByName))
	assert.Equal(t, 10, len(pc.nameByID))
	val, ok := pc.GetByID(1)
	assert.True(t, ok)
	assert.Equal(t, "kernel", val.Name)
	assert.Equal(t, "5.6.13-200.fc31.x86_64", val.Evra)
	assert.Equal(t, 101, val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
	val, ok = pc.GetByNevra("kernel-0:5.6.13-201.fc31.x86_64")
	assert.True(t, ok)
	assert.Equal(t, 11, val.ID)
	assert.Equal(t, "kernel", val.Name)
	assert.Equal(t, "5.6.13-201.fc31.x86_64", val.Evra)
	assert.Equal(t, 101, val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
	val, ok = pc.GetLatestByName("kernel")
	assert.True(t, ok)
	assert.Equal(t, 11, val.ID)
	assert.Equal(t, "kernel", val.Name)
	assert.Equal(t, "5.6.13-201.fc31.x86_64", val.Evra)
	assert.Equal(t, 101, val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
	valStr, ok := pc.GetNameByID(104)
	assert.True(t, ok)
	assert.Equal(t, "curl", valStr)
}

func TestAddPackageCache(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	pc := NewPackageCache()
	assert.NotNil(t, pc)
	pc.Load()
	assert.Equal(t, 12, len(pc.byID))
	assert.Equal(t, 12, len(pc.byNevra))
	assert.Equal(t, 10, len(pc.latestByName))
	assert.Equal(t, 10, len(pc.nameByID))

	pkg := PackageCacheMetadata{
		ID:              999,
		NameID:          104,
		Name:            "curl",
		Evra:            "999-999.x86_64",
		DescriptionHash: []byte("44"),
		SummaryHash:     []byte("4"),
	}
	pc.Add(&pkg)
	assert.Equal(t, 13, len(pc.byID))
	assert.Equal(t, 13, len(pc.byNevra))
	assert.Equal(t, 10, len(pc.latestByName))
	assert.Equal(t, 10, len(pc.nameByID))

	val, ok := pc.GetByID(999)
	assert.True(t, ok)
	assert.Equal(t, pkg.Name, val.Name)
	assert.Equal(t, pkg.Evra, val.Evra)
	assert.Equal(t, pkg.NameID, val.NameID)
	assert.Equal(t, pkg.DescriptionHash, val.DescriptionHash)
	assert.Equal(t, pkg.SummaryHash, val.SummaryHash)
}
