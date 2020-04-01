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
	updateExprs := make([]UpExpr, len(updateCols))
	for i, v := range updateCols {
		updateExprs[i] = UpExpr{v, fmt.Sprintf("excluded.%s", v)}
	}
	return OnConflictDoUpdateExpr(db, keys, updateExprs...)
}

type UpExpr struct {
	Name string
	Expr string
}

func OnConflictDoUpdateExpr(db *gorm.DB, keys []string, updateExprs ...UpExpr) *gorm.DB {
	keyStr := strings.Join(keys, ",")
	updateStrs := make([]string, len(updateExprs))
	for i, v := range updateExprs {
		val := fmt.Sprintf("%s = %s", v.Name, v.Expr)
		updateStrs[i] = val
	}
	valStr := strings.Join(updateStrs, ",")
	if valStr != "" {
		option := fmt.Sprintf("ON CONFLICT (%v) DO UPDATE SET %v", keyStr, valStr) // nolint:gosec
		return db.Set("gorm:insert_option", option)
	}
	return db
}
