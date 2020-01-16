package database

type TestTable struct {
	ID    uint `gorm:"primary_key"`
	Name  string
	Email string
}

type TestTableSlice []TestTable

func (t TestTableSlice) MakeInterfaceSlice() []interface{} {
	res := make([]interface{}, len(t))
	for i := range t {
		res[i] = t[i]
	}
	return res
}
