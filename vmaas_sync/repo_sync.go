package vmaas_sync //nolint:revive,stylecheck
import (
	"app/base/database"
	"github.com/pkg/errors"
	"time"
)

func syncRepos(syncStart time.Time, modifiedSince *string) error {
	// mark non-thirdparty repos known to vmaas
	updateRepos, err := getUpdatedRepos(syncStart, modifiedSince, false)
	if err != nil {
		return err
	}

	if len(updateRepos) == 0 {
		return nil
	}

	err = database.Db.Exec("UPDATE repo SET third_party = false WHERE name in (?)", updateRepos).Error
	if err != nil {
		return errors.WithMessage(err, "Updating repo third_party flag")
	}
	return nil
}
