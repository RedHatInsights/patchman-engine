package database

type TestTable struct {
	ID    uint `gorm:"primary_key"`
	Name  string
	Email string
}

type TestTableSlice []TestTable
