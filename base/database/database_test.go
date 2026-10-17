package database

import (
	"app/base/utils"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnConflictDoUpdate(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()
	db := TestDB()
	err := db.AutoMigrate(&TestTable{})
	assert.NoError(t, err)
	err = db.Unscoped().Delete(&TestTable{}, "true").Error
	assert.NoError(t, err)

	obj := TestTable{
		Name:  "Bla",
		Email: "Bla",
	}

	assert.Equal(t, nil, OnConflictUpdate(db, "id", "name", "email").Create(&obj).Error)

	var read TestTable
	db.Find(&read, obj.ID)

	assert.Equal(t, obj.ID, read.ID)
	assert.Equal(t, obj.Name, read.Name)
	assert.Equal(t, obj.Email, read.Email)

	obj.Name = ""

	assert.Equal(t, nil, OnConflictUpdate(db, "id", "name", "email").Create(&obj).Error)

	db.Find(&read, obj.ID)

	assert.Equal(t, obj.ID, read.ID)
	assert.Equal(t, obj.Name, read.Name)
	assert.Equal(t, obj.Email, read.Email)
}

func TestCancelContext(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	ctx, cancel := context.WithCancel(context.Background())
	tx := DB(ctx).Begin()
	cancel()
	err := tx.Exec("select pg_sleep(1)").Error
	assert.NotNil(t, err)
	assert.Equal(t, "context canceled", err.Error())
}

func TestStatementTimeout(t *testing.T) {
	utils.CoreCfg.DBStatementTimeoutMs = 100
	utils.SkipWithoutDB(t)
	Configure()

	err := TestDB().Exec("select pg_sleep(10)").Error
	assert.NotNil(t, err)
	assert.Equal(t, "ERROR: canceling statement due to statement timeout (SQLSTATE 57014)", err.Error())
}
