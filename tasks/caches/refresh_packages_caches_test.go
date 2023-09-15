package caches

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

// save counts from getCounts as global for other tests
var (
	_counts = make([]models.PackageAccountData, 0)
	_acc    = 3
)

func TestAccountsWithoutCache(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	accs, err := accountsWithoutCache()
	assert.Nil(t, err)
	// there are only 4 account in test_data but other tests are creating new accounts
	assert.Equal(t, 10, len(accs))
}

func TestGetCounts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	err := getCounts(&_counts, &_acc)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(_counts))
}

func TestUpdatePackageAccountData(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	err := updatePackageAccountData(_counts)
	assert.Nil(t, err)

	// delete old cache data, just check it does not return error
	err = deleteOldCache(_counts, &_acc)
	assert.Nil(t, err)
}

func TestUpdatePkgCacheValidity(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	err := updatePkgCacheValidity(&_acc)
	assert.Nil(t, err)

	// set back to false
	database.Db.Table("rh_account").
		Where("id = ?", _acc).
		Update("valid_package_cache = ?", false)
}
