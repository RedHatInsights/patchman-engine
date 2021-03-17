package vmaas_sync //nolint:golint,stylecheck
import (
	"app/base/database"
	"github.com/pkg/errors"
)

func syncRepos() error {
	// mark all repos known to vmaas as our (i.e. not thirdparty)
	updateRepos, err := getUpdatedRepos(nil)
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
