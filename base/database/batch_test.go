package database

import (
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

var defaultValues = TestTableSlice{
	{Name: "A", Email: "B",},
	{Name: "C", Email: "D",},
	{Name: "E", Email: "F",},
	{Name: "G", Email: "H",},
	{Name: "I", Email: "J",},
	{Name: "K", Email: "L",},
	{Name: "M", Email: "N",},
}

func TestBatchInsert(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	Db.AutoMigrate(&TestTable{})
	Db.Unscoped().Delete(&TestTable{})

	arr := defaultValues.MakeInterfaceSlice()

	err := BulkInsert(Db, arr)
	assert.Nil(t, err)

	var res []TestTable
	assert.Nil(t, Db.Find(&res).Error)

	assert.Equal(t, len(defaultValues), len(res))
	for i := range defaultValues {
		assert.Equal(t, res[i].Name, defaultValues[i].Name)
		assert.Equal(t, res[i].Email, defaultValues[i].Email)
	}
}

func TestBatchInsertChunked(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	Db.AutoMigrate(&TestTable{})
	Db.Unscoped().Delete(&TestTable{})

	arr := defaultValues.MakeInterfaceSlice()

	err := BulkInsertChunk(Db, arr, 2)
	assert.Nil(t, err)

	var res []TestTable
	assert.Nil(t, Db.Find(&res).Error)

	assert.Equal(t, len(defaultValues), len(res))
	for i := range defaultValues {
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

	arr := defaultValues.MakeInterfaceSlice()

	// Perform first insert
	err := BulkInsert(db, arr)
	assert.Nil(t, err)

	var outputs []TestTable
	assert.Nil(t, db.Find(&outputs).Error)

	assert.Equal(t, len(defaultValues), len(outputs))
	for i := range defaultValues {
		assert.Equal(t, defaultValues[i].Name, outputs[i].Name)
		assert.Equal(t, defaultValues[i].Email, outputs[i].Email)

		outputs[i].Name = ""
	}
	arr = TestTableSlice(outputs).MakeInterfaceSlice()

	// Try to re-insert, and update values
	db = OnConflictUpdate(db, "id", "name", "email")
	err = BulkInsert(db, arr)
	assert.Nil(t, err)

	// Re-load data from database
	var modified []TestTable
	assert.Nil(t, db.Find(&modified).Error)

	for i := range outputs {
		assert.Equal(t, outputs[i].Name, modified[i].Name)
		assert.Equal(t, outputs[i].Email, modified[i].Email)
	}

}
