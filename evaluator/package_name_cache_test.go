package evaluator

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetNameIDHashesMapCache(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	ConfigurePackageNameCache()
	assert.NotNil(t, packageNameCacheData)
	assert.Equal(t, 10, len(packageNameCacheData.data))
	val, ok := GetPackageNameMetadata("kernel")
	assert.True(t, ok)
	assert.Equal(t, 101, val.NameID)
	assert.Equal(t, "1", string(val.SummaryHash))
	assert.Equal(t, "11", string(val.DescriptionHash))
}
