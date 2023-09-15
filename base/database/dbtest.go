package database

type TestTable struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"unique"`
	Email string
}

type TestTableSlice []TestTable
