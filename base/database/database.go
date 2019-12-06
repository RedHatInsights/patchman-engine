package database

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"app/base/models"
)

// database cleaning method
func DeleteAllRhAccounts() error {
	err := Db.Delete(models.RhAccount{}).Error
	return err
}
