package tasks

import (
	"app/base"
	"app/base/database"
	"app/base/utils"
	"os"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func HandleContextCancel(fn func()) {
	go func() {
		<-base.Context.Done()
		utils.LogInfo("stopping vmaas_sync")
		fn()
	}()
}

func WaitAndExit() {
	time.Sleep(time.Second) // give some time to close eventual db connections
	os.Exit(0)
}

// return database handler with base context
// which will be properly cancled in case of service shutdown
func CancelableDB() *gorm.DB {
	return database.Db.WithContext(base.Context)
}

// return read replica (if available) database handler with base context
// which will be properly canceled in case of service shutdown
func CancelableReadReplicaDB() *gorm.DB {
	if utils.Cfg.DBReadReplicaEnabled && database.ReadReplicaConfigured() {
		return database.DbReadReplica.WithContext(base.Context)
	}
	return database.Db.WithContext(base.Context)
}

func withTx(do func(db *gorm.DB) error, cancelableDB func() *gorm.DB) error {
	tx := cancelableDB().Begin()
	defer tx.Rollback()
	if err := do(tx); err != nil {
		return err
	}
	return errors.Wrap(tx.Commit().Error, "Commit")
}

// Need to run code within a function, because defer can't be used in loops
func WithTx(do func(db *gorm.DB) error) error {
	return withTx(do, CancelableDB)
}

// Need to run code within a function, because defer can't be used in loops
func WithReadReplicaTx(do func(db *gorm.DB) error) error {
	return withTx(do, CancelableReadReplicaDB)
}
