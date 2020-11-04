package vmaas_sync //nolint:golint,stylecheck
import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"time"
)

const LastEvalRepoBased = "last_eval_repo_based"

func getCurrentRepoBasedInventoryIDs() ([]string, error) {
	lastRepoBaseEval, err := getLastRepobasedEvalTms()
	if err != nil {
		return nil, err
	}

	updateRepos, err := getUpdatedRepos(lastRepoBaseEval)
	if err != nil {
		return nil, err
	}

	inventoryIDs, err := getRepoBasedInventoryIDs(updateRepos)
	if err != nil {
		return nil, err
	}

	updateRepoBaseEvalTimestamp(time.Now())

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

func updateRepoBaseEvalTimestamp(value time.Time) {
	ts := value.Format(time.RFC3339)
	err := updateRepoBaseEvalTimestampStr(ts)
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update repo-based eval. timestamp")
	}
}

func updateRepoBaseEvalTimestampStr(value string) error {
	err := database.Db.Exec("INSERT INTO timestamp_kv (name, value) values (?, ?)"+
		"ON CONFLICT (name) DO UPDATE SET value = ?", LastEvalRepoBased, value, value).Error
	return err
}

func getRepoBasedInventoryIDs(repos []string) ([]string, error) {
	var intentoryIDs []string
	if len(repos) == 0 {
		return intentoryIDs, nil
	}

	err := database.Db.Table("system_repo sr").
		Joins("JOIN repo ON repo.id = sr.repo_id").
		Joins("JOIN system_platform sp ON sp.id = sr.system_id").
		Where("repo.name IN (?)", repos).
		Order("inventory_id ASC").
		Pluck("distinct inventory_id", &intentoryIDs).Error
	if err != nil {
		return nil, err
	}
	return intentoryIDs, nil
}

//nolint unused
func getUpdatedRepos(modifiedSince *time.Time) ([]string, error) {
	page := float32(1)
	var reposArr []string
	for {
		reposReq := vmaas.ReposRequest{
			Page:           page,
			RepositoryList: []string{".*"},
			PageSize:       float32(advisoryPageSize),
		}

		if modifiedSince != nil {
			reposReq.ModifiedSince = modifiedSince.Format(time.RFC3339)
		}

		vmaasCallArgs := vmaas.AppReposHandlerPostPostOpts{
			ReposRequest: optional.NewInterface(reposReq),
		}

		repos, _, err := vmaasClient.DefaultApi.AppReposHandlerPostPost(base.Context, &vmaasCallArgs)
		if err != nil {
			return nil, err
		}
		vmaasCallCnt.WithLabelValues("success").Inc()

		if repos.Pages == 0 {
			utils.Log().Debug("No repos returned from VMaaS")
			break
		}

		utils.Log("count", len(repos.RepositoryList)).Debug("Downloaded repos")
		for k, _ := range repos.RepositoryList {
			reposArr = append(reposArr, k)
		}

		if page == repos.Pages {
			break
		}
		page++
	}

	utils.Log("count", len(reposArr)).Info("Repos downloading complete")
	return reposArr, nil
}
