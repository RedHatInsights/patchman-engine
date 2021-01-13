package database

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Appends `ON CONFLICT (key...) DO UPDATE SET (fields) to following insert query
func OnConflictUpdate(db *gorm.DB, key string, updateCols ...string) *gorm.DB {
	return OnConflictUpdateMulti(db, []string{key}, updateCols...)
}

// Appends `ON CONFLICT (key...) DO UPDATE SET (fields) to following insert query with multiple key fields
func OnConflictUpdateMulti(db *gorm.DB, keys []string, updateCols ...string) *gorm.DB {
	confilctColumns := []clause.Column{}
	for _, key := range keys {
		confilctColumns = append(confilctColumns, clause.Column{Name: key})
	}
	return db.Clauses(clause.OnConflict{
		Columns:   confilctColumns,
		DoUpdates: clause.AssignmentColumns(updateCols),
	})
}

type UpExpr struct {
	Name string
	Expr string
}

func OnConflictDoUpdateExpr(db *gorm.DB, keys []string, updateExprs ...UpExpr) *gorm.DB {
	updateColsValues := make(map[string]interface{})
	for _, v := range updateExprs {
		updateColsValues[v.Name] = v.Expr
	}
	confilctColumns := []clause.Column{}
	for _, key := range keys {
		confilctColumns = append(confilctColumns, clause.Column{Name: key})
	}
	if updateColsValues != nil {
		// option := fmt.Sprintf("ON CONFLICT (%v) DO UPDATE SET %v", keyStr, valStr) // nolint:gosec
		// return db.Set("gorm:insert_option", option)
		return db.Clauses(clause.OnConflict{
			Columns:   confilctColumns,
			DoUpdates: clause.Assignments(updateColsValues),
		})
	}
	return db
}
