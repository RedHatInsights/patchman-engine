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
		utils.Log().Info("stopping vmaas_sync")
		fn()
	}()
}

func WaitAndExit() {
	time.Sleep(time.Second) // give some time to close eventual db connections
	os.Exit(0)
}

// Need to run code within a function, because defer can't be used in loops
func WithTx(do func(db *gorm.DB) error) error {
	tx := database.Db.WithContext(base.Context).Begin()
	defer tx.Rollback()
	if err := do(tx); err != nil {
		return err
	}
	return errors.Wrap(tx.Commit().Error, "Commit")
}
