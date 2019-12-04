package database

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"app/base/structures"
)

// database cleaning method
func DelteAllRhAccounts() error {
	err := Db.Delete(structures.RhAccountDAO{}).Error
	return err
}

func HostsCount() (int, error) {
	cnt := 0
	err := Db.Model(structures.RhAccountDAO{}).Count(&cnt).Error
	return cnt, err
}
