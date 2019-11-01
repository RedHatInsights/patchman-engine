package database

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"gin-container/app/structures"
)

// database cleaning method
func DelteAllHosts() error {
	err := Db.Delete(structures.HostDAO{}).Error
	return err
}

func HostsCount() (int, error) {
	cnt := 0
	err := Db.Model(structures.HostDAO{}).Count(&cnt).Error
	return cnt, err
}
