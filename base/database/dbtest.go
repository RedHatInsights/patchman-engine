package database

type TestTable struct {
	ID    uint `gorm:"primary_key"`
	Name  string
	Email string
}

type TestTableSlice []TestTable

func (this TestTableSlice) MakeInterfaceSlice() []interface{} {
	res := make([]interface{}, len(this))
	for i := range this {
		res[i] = this[i]
	}
	return res
}
