package vmaas_sync //nolint:golint,stylecheck
import (
	"app/base/database"
	"github.com/pkg/errors"
	"time"
)

func syncRepos(modifiedSince *time.Time) error {
	// mark non-thirdparty repos known to vmaas
	updateRepos, err := getUpdatedRepos(modifiedSince, false)
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
