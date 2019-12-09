package database

import "github.com/jinzhu/gorm"

type TestTable struct {
	Name  string
	Email string
	gorm.Model
}

type TestTableSlice []TestTable
func (this TestTableSlice) MakeInterfaceSlice() []interface{} {
	res := make([]interface{}, len(this))
	for i := range this {
		res[i] = this[i]
	}
	return res
}
