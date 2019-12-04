package listener

import (
	"app/base/database"
	"app/base/structures"
	"app/base/utils"
	"github.com/bmizerany/assert"
	"testing"

	"app/base/core"
)

func TestStorageInit(t *testing.T) {
	storage := InitStorage(3, false)
	assert.Equal(t, 0, storage.StoredItems())
	assert.Equal(t, 3, storage.Capacity())
}

func TestStorageFlush(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	storage := InitStorage(3, false)

	for _, item := range []structures.RhAccountDAO{{1, "1"}, {2, "2"}} {
		err := storage.Add(&item)
		assert.Equal(t, nil, err)
	}
	assert.Equal(t, 2, storage.StoredItems())
	assert.Equal(t, 3, storage.Capacity())

	err := storage.Flush() // write items to database
	assert.Equal(t, nil, err)

	// ensure items in database
	cnt := 0
	database.Db.Model(&structures.RhAccountDAO{}).Count(&cnt)
	assert.Equal(t, 2, cnt)
}

func TestStorageBuffer(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	storage := InitStorage(2, false)

	for _, item := range []structures.RhAccountDAO{{1, "1"}, {2, "2"}, {3, "3"}} {
		err := storage.Add(&item)
		assert.Equal(t, nil, err)
	}
	assert.Equal(t, 1, storage.StoredItems())
	assert.Equal(t, 2, storage.Capacity())

	// ensure items in database
	cnt := 0
	database.Db.Model(&structures.RhAccountDAO{}).Count(&cnt)
	assert.Equal(t, 2, cnt)
}
