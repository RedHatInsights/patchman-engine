package database

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AutonomousTx starts a transaction on the root connection pool.
// The caller may already be inside a transaction, and beginning another one
// on that handle returns gorm.ErrInvalidTransaction. This transaction commits
// independently of the caller.
func AutonomousTx(tx *gorm.DB) *gorm.DB {
	ctx := context.Background()
	if tx.Statement != nil && tx.Statement.Context != nil {
		ctx = tx.Statement.Context
	}
	session := tx.Session(&gorm.Session{NewDB: true, Context: ctx, Initialized: true})
	session.Statement.ConnPool = tx.ConnPool
	return session.Begin()
}

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

type UpExpr struct {
	Name string
	Expr string
}

func OnConflictDoUpdateExpr(db *gorm.DB, keys []string, updateExprs ...UpExpr) *gorm.DB {
	updateColsValues := make(map[string]interface{}, len(updateExprs))
	for _, v := range updateExprs {
		updateColsValues[v.Name] = v.Expr
	}
	conflictColumns := make([]clause.Column, len(keys))
	for i, key := range keys {
		conflictColumns[i] = clause.Column{Name: key}
	}
	if len(updateColsValues) > 0 {
		return db.Clauses(clause.OnConflict{
			Columns:   conflictColumns,
			DoUpdates: clause.Assignments(updateColsValues),
		})
	}
	return db
}
