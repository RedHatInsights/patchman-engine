package listener

import (
	"gin-container/app/database"
	"gin-container/app/structures"
	"github.com/bmizerany/assert"
	"testing"

	"gin-container/app/core"
)

func TestStorageInit(t *testing.T) {
	storage := InitStorage(3)
	assert.Equal(t, 0, storage.StoredItems())
	assert.Equal(t, 3, storage.Capacity())
}

func TestStorageFlush(t *testing.T) {
	core.SetupTestEnvironment()
	storage := InitStorage(3)

	for _, item := range []structures.HostDAO{{ID: 1}, {ID: 2}} {
		err := storage.Add(&item)
		assert.Equal(t, nil, err)
	}
	assert.Equal(t, 2, storage.StoredItems())
	assert.Equal(t, 3, storage.Capacity())

	err := storage.Flush() // write items to database
	assert.Equal(t, nil, err)

	// ensure items in database
	cnt := 0
	database.Db.Model(&structures.HostDAO{}).Count(&cnt)
	assert.Equal(t, 2, cnt)
}

func TestStorageBuffer(t *testing.T) {
	core.SetupTestEnvironment()
	storage := InitStorage(2)

	for _, item := range []structures.HostDAO{{ID: 1}, {ID: 2}, {ID: 3}} {
		err := storage.Add(&item)
		assert.Equal(t, nil, err)
	}
	assert.Equal(t, 1, storage.StoredItems())
	assert.Equal(t, 2, storage.Capacity())

	// ensure items in database
	cnt := 0
	database.Db.Model(&structures.HostDAO{}).Count(&cnt)
	assert.Equal(t, 2, cnt)
}
