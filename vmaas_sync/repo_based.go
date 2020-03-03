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

func getRepoBasedInventoryIDs(repos []string) (*[]string, error) {
	var intentoryIDs []string
	err := database.Db.Table("system_repo sr").
		Joins("JOIN repo ON repo.id = sr.repo_id").
		Joins("JOIN system_platform sp ON sp.id = sr.system_id").
		Where("repo.name IN (?)", repos).
		Order("inventory_id ASC").
		Pluck("inventory_id", &intentoryIDs).Error
	if err != nil {
		return nil, err
	}
	return &intentoryIDs, nil
}
