package vmaas_sync //nolint:golint,stylecheck
import (
	"app/base/database"
	"app/base/models"
	"context"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"time"
)

const LastEvalRepoBased = "last_eval_repo_based"

func getCurrentRepoBasedInventoryIDs() (*[]string, error) {
	lastRepoBaseEval, err := getLastRepobasedEvalTms()
	if err != nil {
		return nil, err
	}

	updateRepos, err := getUpdatedRepos(*lastRepoBaseEval)
	if err != nil {
		return nil, err
	}

	inventoryIDs, err := getRepoBasedInventoryIDs(updateRepos)
	if err != nil {
		return nil, err
	}
	return inventoryIDs, nil
}

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

func getRepoBasedInventoryIDs(repos *[]string) (*[]string, error) {
	var intentoryIDs []string
	err := database.Db.Table("system_repo sr").
		Joins("JOIN repo ON repo.id = sr.repo_id").
		Joins("JOIN system_platform sp ON sp.id = sr.system_id").
		Where("repo.name IN (?)", *repos).
		Order("inventory_id ASC").
		Pluck("inventory_id", &intentoryIDs).Error
	if err != nil {
		return nil, err
	}
	return &intentoryIDs, nil
}

//nolint unused
func getUpdatedRepos(modifiedSince time.Time) (*[]string, error){
	page := float32(1)
	var reposArr []string
	for {
		reposReq := vmaas.ReposRequest{
			Page:           page,
			RepositoryList: []string{".*"},
			ModifiedSince:  modifiedSince.Format(time.RFC3339),
		}

		vmaasCallArgs := vmaas.AppReposHandlerPostPostOpts{
			ReposRequest: optional.NewInterface(reposReq),
		}

		repos, _, err := vmaasClient.ReposApi.AppReposHandlerPostPost(context.Background(), &vmaasCallArgs)
		if err != nil {
			return nil, err
		}

		for k, _ := range repos.RepositoryList {
			reposArr = append(reposArr, k)
		}

		if page == repos.Pages {
			break
		}
		page++
	}

	return &reposArr, nil
}
