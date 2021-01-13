package database

import (
	"app/base"
	"app/base/utils"
	// "context"
	"github.com/stretchr/testify/assert"
	// "gorm.io/gorm"
	"os"
	"testing"
	"time"
)

func TestOnConflictDoUpdate(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	_ = Db.AutoMigrate(&TestTable{})
	Db.Unscoped().Exec("DELETE FROM test_tables")

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

	tx := Db.WithContext(base.Context)
	go func() {
		time.Sleep(time.Millisecond * 100)
		base.CancelContext()
	}()
	base.CancelContext()
	err := tx.Exec("select pg_sleep(1)").Error
	assert.NotNil(t, err)
	assert.Equal(t, "context canceled", err.Error())
}

func TestStatementTimeout(t *testing.T) {
	assert.Nil(t, os.Setenv("DB_STATEMENT_TIMEOUT_MS", "100"))
	utils.SkipWithoutDB(t)
	Configure()

	err := Db.Exec("select pg_sleep(10)").Error
	assert.NotNil(t, err)
	assert.Equal(t, "ERROR: canceling statement due to statement timeout (SQLSTATE 57014)", err.Error())
}
