package database

import (
	"app/base/utils"
	"github.com/bmizerany/assert"
	"testing"
)

func TestBatchInsert(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	Db.AutoMigrate(&TestTable{})
	Db.Unscoped().Delete(&TestTable{})

	vals := TestTableSlice{
		{
			Name:  "A",
			Email: "B",
		},
		{
			Name:  "A",
			Email: "B",
		},
		{
			Name:  "A",
			Email: "B",
		},
	}
	arr := vals.MakeInterfaceSlice()

	err := BulkInsert(Db, arr)
	assert.Equal(t, nil, err)

	var res []TestTable
	assert.Equal(t, nil, Db.Find(&res).Error)

	assert.Equal(t, len(vals), len(res))
	for i := range vals {
		assert.Equal(t, res[i].Name, vals[i].Name)
		assert.Equal(t, res[i].Email, vals[i].Email)
	}
}

func TestBatchInsertOnConflictUpdate(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()
	db := Db

	db.AutoMigrate(&TestTable{})
	db.Unscoped().Delete(&TestTable{}, "true")

	inputs := TestTableSlice{
		{
			Name:  "A",
			Email: "B",
		},
		{
			Name:  "A",
			Email: "B",
		},
		{
			Name:  "A",
			Email: "B",
		},
	}

	arr := inputs.MakeInterfaceSlice()

	// Perform first insert
	err := BulkInsert(db, arr)
	assert.Equal(t, nil, err)

	var outputs []TestTable
	assert.Equal(t, nil, db.Find(&outputs).Error)

	assert.Equal(t, len(inputs), len(outputs))
	for i := range inputs {
		assert.Equal(t, inputs[i].Name, outputs[i].Name)
		assert.Equal(t, inputs[i].Email, outputs[i].Email)

		outputs[i].Name = ""
	}
	arr = TestTableSlice(outputs).MakeInterfaceSlice()

	// Try to re-insert, and update values
	db = OnConflictUpdate(db, "id", "name", "email")
	err = BulkInsert(db, arr)
	assert.Equal(t, nil, err)

	// Re-load data from database
	var modified []TestTable
	assert.Equal(t, nil, db.Find(&modified).Error)

	for i := range outputs {
		assert.Equal(t, outputs[i].Name, modified[i].Name)
		assert.Equal(t, outputs[i].Email, modified[i].Email)
	}

}
