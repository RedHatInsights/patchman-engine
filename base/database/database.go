package database

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"strings"
)

// Appends `ON CONFLICT (key...) DO UPDATE SET (fields) to following insert query
func OnConflictUpdate(db *gorm.DB, key string, updateCols ...string) *gorm.DB {
	return OnConflictUpdateMulti(db, []string{key}, updateCols...)
}

// Appends `ON CONFLICT (key...) DO UPDATE SET (fields) to following insert query with multiple key fields
func OnConflictUpdateMulti(db *gorm.DB, keys []string, updateCols ...string) *gorm.DB {
	keyStr := strings.Join(keys, ",")
	updateExprs := make([]string, len(updateCols))
	for i, v := range updateCols {
		val := fmt.Sprintf("%v = excluded.%v", v, v)
		updateExprs[i] = val
	}
	valStr := strings.Join(updateExprs, ",")
	if valStr != "" {
		option := fmt.Sprintf("ON CONFLICT (%v) DO UPDATE SET %v", keyStr, valStr)
		return db.Set("gorm:insert_option", option)
	} else {
		return db
	}
}
