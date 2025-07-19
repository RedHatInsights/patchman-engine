package vmaas_sync

import (
	"app/tasks"
	"time"

	"github.com/pkg/errors"
)

func syncRepos(syncStart time.Time) error {
	// mark non-thirdparty repos known to vmaas
	thirdParty := false
	repoPackages, repoNoPackages, _, err := getUpdatedRepos(syncStart, nil, &thirdParty)
	if err != nil {
		return err
	}

	redhatRepos := repoNoPackages
	for _, repoPkg := range repoPackages {
		redhatRepos = append(redhatRepos, repoPkg[0])
	}

	if len(redhatRepos) == 0 {
		return nil
	}

	err = tasks.CancelableDB().Exec("UPDATE repo SET third_party = false WHERE name in (?)", redhatRepos).Error
	if err != nil {
		return errors.WithMessage(err, "Updating repo third_party flag for redhat content")
	}

	err = tasks.CancelableDB().Exec("UPDATE repo SET third_party = true WHERE name NOT IN (?)", redhatRepos).Error
	if err != nil {
		return errors.WithMessage(err, "Updating repo third_party flag for third party content")
	}
	return nil
}
