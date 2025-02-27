package database

import (
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestBatchInsert(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	err := DB.AutoMigrate(&TestTable{})
	assert.NoError(t, err)
	err = DB.Unscoped().Delete(&TestTable{}, "true").Error
	assert.NoError(t, err)

	// Bulk insert should create new rows
	values := make(TestTableSlice, len(defaultValues))
	copy(values, defaultValues)
	err = BulkInsert(DB, values, WithReturning("RETURNING *"))
	assert.NoError(t, err)

	var res []TestTable
	assert.NoError(t, DB.Find(&res).Error)

	// Reading rows should return same data as the inserted rows
	assert.Equal(t, len(values), len(res))
	for i := range values {
		assert.Equal(t, res[i].ID, values[i].ID)
		assert.Equal(t, res[i].Name, values[i].Name)
		assert.Equal(t, res[i].Email, values[i].Email)
	}
}

func TestBatchInsertOnConflictUpdate(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()
	db := DB

	err := DB.AutoMigrate(&TestTable{})
	assert.NoError(t, err)
	err = DB.Unscoped().Delete(&TestTable{}, "true").Error
	assert.NoError(t, err)

	// Perform first insert
	values := make(TestTableSlice, len(defaultValues))
	copy(values, defaultValues)
	err = BulkInsert(db, values, WithReturning("RETURNING *"))
	assert.NoError(t, err)

	var outputs []TestTable
	assert.NoError(t, db.Find(&outputs).Error)

	assert.Equal(t, len(values), len(outputs))
	for i := range values {
		assert.Equal(t, values[i].ID, outputs[i].ID)
		assert.Equal(t, values[i].Name, outputs[i].Name)
		assert.Equal(t, values[i].Email, outputs[i].Email)
		// Clear ids
		outputs[i].ID = 0
		outputs[i].Email = ""
	}

	// Try to re-insert, and update values
	db = OnConflictUpdate(db, "name", "name", "email")
	err = BulkInsert(db, outputs, WithReturning("RETURNING *"))
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

func TestBatchInsertNoReturning(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	err := DB.AutoMigrate(&TestTable{})
	assert.NoError(t, err)
	err = DB.Unscoped().Delete(&TestTable{}, "true").Error
	assert.NoError(t, err)

	// Bulk insert should create new rows
	values := make(TestTableSlice, len(defaultValues))
	copy(values, defaultValues)
	err = BulkInsert(DB, values)
	assert.NoError(t, err)

	var res []TestTable
	assert.NoError(t, DB.Find(&res).Error)

	// Reading rows should return same data as the inserted rows
	assert.Equal(t, len(values), len(res))
	for i := range values {
		assert.Equal(t, uint(0), values[i].ID) // ID is not populated due to returning=false
		assert.NotEqual(t, values[i].ID, res[i].ID)
		assert.Equal(t, values[i].Name, res[i].Name)
		assert.Equal(t, values[i].Email, res[i].Email)
	}
}
