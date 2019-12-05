package database

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"app/base/structures"
)

// database cleaning method
func DeleteAllRhAccounts() error {
	err := Db.Delete(structures.RhAccountDAO{}).Error
	return err
}
