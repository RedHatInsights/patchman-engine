package vmaas_sync //nolint:golint,stylecheck
import (
	"app/base/database"
	"app/base/models"
	"time"
)

const LastEvalRepoBased = "last_eval_repo_based"

func getLastRepobasedEvalTms() (*time.Time, error) {
	var timestamps []*time.Time
	err := database.Db.Model(&models.TimestampKV{}).
		Where("name = ?", LastEvalRepoBased).
		Pluck("value", &timestamps).Error
	if err != nil {
		return nil, err
	}

	if len(timestamps) == 0 {
		return nil, nil
	}
	return timestamps[0], nil
}
