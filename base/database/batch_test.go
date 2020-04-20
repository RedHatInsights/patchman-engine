package database

import (
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

var defaultValues = TestTableSlice{
	{Name: "A", Email: "B"},
	{Name: "C", Email: "D"},
	{Name: "E", Email: "F"},
	{Name: "G", Email: "H"},
	{Name: "I", Email: "J"},
	{Name: "K", Email: "L"},
	{Name: "M", Email: "N"},
}

// Bulk insert should only accept slices not raw values
func TestBatchInsertTypes(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()
	values := []interface{}{0, "", &defaultValues[0]}

	for _, v := range values {
		assert.Error(t, BulkInsert(Db, v))
	}

	for _, v := range values {
		assert.Error(t, BulkInsertChunk(Db, v, 10)[0])
	}

	arr := []TestTable{}
	assert.Error(t, BulkInsert(Db, &arr))
	assert.Error(t, BulkInsertChunk(Db, &arr, 10)[0])

	wrongTypeArr := []int{0}
	assert.Error(t, BulkInsert(Db, &wrongTypeArr))
	assert.Error(t, BulkInsertChunk(Db, &wrongTypeArr, 10)[0])
}

func TestBatchInsert(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	Db.AutoMigrate(&TestTable{})
	Db.Unscoped().Delete(&TestTable{})

	// Bulk insert should create new rows
	err := BulkInsert(Db, defaultValues)
	assert.NoError(t, err)

	var res []TestTable
	assert.NoError(t, Db.Find(&res).Error)

	// Reading rows should return same data as the inserted rows
	assert.Equal(t, len(defaultValues), len(res))
	for i := range defaultValues {
		assert.Equal(t, res[i].ID, defaultValues[i].ID)
		assert.Equal(t, res[i].Name, defaultValues[i].Name)
		assert.Equal(t, res[i].Email, defaultValues[i].Email)
	}
}

func TestBatchInsertChunked(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	Db.AutoMigrate(&TestTable{})
	Db.Unscoped().Delete(&TestTable{})

	err := BulkInsertChunk(Db, defaultValues, 2)
	assert.Nil(t, err)

	var res []TestTable
	assert.NoError(t, Db.Find(&res).Error)

	// Same behavior as before, chunked save should scan database values into the source slice
	assert.Equal(t, len(defaultValues), len(res))
	for i := range defaultValues {
		assert.Equal(t, res[i].ID, defaultValues[i].ID)
		assert.Equal(t, res[i].Name, defaultValues[i].Name)
		assert.Equal(t, res[i].Email, defaultValues[i].Email)
	}
}

func TestBatchInsertOnConflictUpdate(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()
	db := Db

	db.AutoMigrate(&TestTable{})
	db.Unscoped().Delete(&TestTable{}, "true")

	// Perform first insert
	err := BulkInsert(db, defaultValues)
	assert.NoError(t, err)

	var outputs []TestTable
	assert.NoError(t, db.Find(&outputs).Error)

	assert.Equal(t, len(defaultValues), len(outputs))
	for i := range defaultValues {
		assert.Equal(t, defaultValues[i].ID, outputs[i].ID)
		assert.Equal(t, defaultValues[i].Name, outputs[i].Name)
		assert.Equal(t, defaultValues[i].Email, outputs[i].Email)
		// Clear ids
		outputs[i].ID = 0
		outputs[i].Email = ""
	}

	// Try to re-insert, and update values
	db = OnConflictUpdate(db, "name", "name", "email")
	err = BulkInsert(db, outputs)
	assert.NoError(t, err)

	// Re-load data from database
	var final []TestTable
	assert.NoError(t, db.Find(&final).Error)

	// Final data should match updated data
	for i := range outputs {
		assert.Equal(t, outputs[i].ID, final[i].ID)
		assert.Equal(t, outputs[i].Name, final[i].Name)
		assert.Equal(t, outputs[i].Email, final[i].Email)
		assert.Equal(t, "", outputs[i].Email)
	}
}
