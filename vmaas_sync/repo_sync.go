package vmaas_sync //nolint:revive,stylecheck
import (
	"app/base/database"
	"github.com/pkg/errors"
	"time"
)

func syncRepos(syncStart time.Time, modifiedSince *string) error {
	// mark non-thirdparty repos known to vmaas
	redhatRepos, _, err := getUpdatedRepos(syncStart, modifiedSince)
	if err != nil {
		return err
	}

	if len(redhatRepos) == 0 {
		return nil
	}

	err = database.Db.Exec("UPDATE repo SET third_party = false WHERE name in (?)", redhatRepos).Error
	if err != nil {
		return errors.WithMessage(err, "Updating repo third_party flag for redhat content")
	}

	err = database.Db.Exec("UPDATE repo SET third_party = true WHERE name NOT IN (?)", redhatRepos).Error
	if err != nil {
		return errors.WithMessage(err, "Updating repo third_party flag for third party content")
	}
	return nil
}
