package database

import (
	"app/base"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"time"

	"testing"
)

func TestOnConflictDoUpdate(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	Db.AutoMigrate(&TestTable{})
	Db.Unscoped().Delete(&TestTable{})

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

	tx := Db.BeginTx(base.Context, nil)

	go func() {
		time.Sleep(time.Millisecond * 100)
		base.CancelContext()
	}()

	err := tx.Exec("select pg_sleep(1)").Error
	assert.NotNil(t, err)
	assert.Equal(t, "pq: canceling statement due to user request", err.Error())
}
