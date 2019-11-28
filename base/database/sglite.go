package database

import (
	"fmt"
	"github.com/satori/go.uuid"
	"app/base/structures"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func ConfigureSQLite() {
	sqliteDbFilePath := fmt.Sprintf("/tmp/app-testing-%s.db", uuid.NewV1().String())

	db, err := gorm.Open("sqlite3", sqliteDbFilePath)
	if err != nil {
		panic(err)
	}
	check(db)

	db.AutoMigrate(&structures.SystemDAO{})

	Db = db
}
