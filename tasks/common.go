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

// Need to run code within a function, because defer can't be used in loops
func WithTx(do func(db *gorm.DB) error) error {
	tx := CancelableDB().Begin()
	defer tx.Rollback()
	if err := do(tx); err != nil {
		return err
	}
	return errors.Wrap(tx.Commit().Error, "Commit")
}
