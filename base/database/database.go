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
	conflictColumns := make([]clause.Column, len(keys))
	for i, key := range keys {
		conflictColumns[i] = clause.Column{Name: key}
	}
	onConflict := clause.OnConflict{Columns: conflictColumns}
	if len(updateCols) > 0 {
		onConflict.DoUpdates = clause.AssignmentColumns(updateCols)
	} else {
		onConflict.DoNothing = true
	}
	return db.Clauses(onConflict)
}
