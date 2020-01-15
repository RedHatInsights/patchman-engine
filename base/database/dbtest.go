package database

type TestTable struct {
	ID    uint   `gorm:"primary_key"`
	Name  string `gorm:"unique"`
	Email string
}

type TestTableSlice []TestTable
