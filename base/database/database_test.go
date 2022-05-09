package database

import (
	"app/base"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnConflictDoUpdate(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	err := Db.AutoMigrate(&TestTable{})
	assert.NoError(t, err)
	err = Db.Unscoped().Delete(&TestTable{}, "true").Error
	assert.NoError(t, err)

	obj := TestTable{
		Name:  "Bla",
		Email: "Bla",
	}

	assert.Equal(t, nil, OnConflictUpdate(Db, "id", "name", "email").Create(&obj).Error)

	var read TestTable
	Db.Find(&read, obj.ID)

	assert.Equal(t, obj.ID, read.ID)
	assert.Equal(t, obj.Name, read.Name)
	assert.Equal(t, obj.Email, read.Email)

	obj.Name = ""

	assert.Equal(t, nil, OnConflictUpdate(Db, "id", "name", "email").Create(&obj).Error)

	Db.Find(&read, obj.ID)

	assert.Equal(t, obj.ID, read.ID)
	assert.Equal(t, obj.Name, read.Name)
	assert.Equal(t, obj.Email, read.Email)
}

func TestCancelContext(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	tx := Db.WithContext(base.Context).Begin()
	base.CancelContext()
	err := tx.Exec("select pg_sleep(1)").Error
	assert.NotNil(t, err)
	assert.Equal(t, "context canceled", err.Error())
}

func TestStatementTimeout(t *testing.T) {
	utils.Cfg.DBStatementTimeoutMs = 100
	utils.SkipWithoutDB(t)
	Configure()

	err := Db.Exec("select pg_sleep(10)").Error
	assert.NotNil(t, err)
	assert.Equal(t, "ERROR: canceling statement due to statement timeout (SQLSTATE 57014)", err.Error())
}
