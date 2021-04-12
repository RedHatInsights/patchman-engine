package vmaas_sync //nolint:golint,stylecheck
import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"time"
)

const LastEvalRepoBased = "last_eval_repo_based"

func getCurrentRepoBasedInventoryIDs() ([]inventoryAID, error) {
	lastRepoBaseEval, err := getLastRepobasedEvalTms()
	if err != nil {
		return nil, err
	}

	updateRepos, err := getUpdatedRepos(lastRepoBaseEval, true)
	if err != nil {
		return nil, err
	}

	inventoryAIDs, err := getRepoBasedInventoryIDs(updateRepos)
	if err != nil {
		return nil, err
	}

	updateRepoBaseEvalTimestamp(time.Now())

	return inventoryAIDs, nil
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

type inventoryAID struct {
	InventoryID string
	RhAccountID int
}

func getRepoBasedInventoryIDs(repos []string) ([]inventoryAID, error) {
	var ids []inventoryAID
	if len(repos) == 0 {
		return ids, nil
	}

	err := database.Db.Table("system_repo sr").
		Joins("JOIN repo ON repo.id = sr.repo_id").
		Joins("JOIN system_platform sp ON  sp.rh_account_id = sr.rh_account_id AND sp.id = sr.system_id").
		Where("repo.name IN (?)", repos).
		Order("sp.rh_account_id").
		Select("distinct sp.inventory_id, sp.rh_account_id").
		Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

//nolint unused
func getUpdatedRepos(modifiedSince *time.Time, thirdParty bool) ([]string, error) {
	page := float32(1)
	var reposArr []string
	for {
		reposReq := vmaas.ReposRequest{
			Page:           vmaas.PtrFloat32(page),
			RepositoryList: []string{".*"},
			PageSize:       vmaas.PtrFloat32(float32(advisoryPageSize)),
			ThirdParty:     vmaas.PtrBool(thirdParty),
		}

		if modifiedSince != nil {
			reposReq.SetModifiedSince(*modifiedSince)
		}

		repos, _, err := vmaasClient.DefaultApi.AppReposHandlerPostPost(base.Context).ReposRequest(reposReq).Execute()
		if err != nil {
			return nil, err
		}
		vmaasCallCnt.WithLabelValues("success").Inc()

		if repos.GetPages() < 1 {
			utils.Log().Debug("No repos returned from VMaaS")
			break
		}

		utils.Log("count", len(repos.GetRepositoryList())).Debug("Downloaded repos")
		for k, _ := range repos.GetRepositoryList() {
			reposArr = append(reposArr, k)
		}

		if page == repos.GetPages() {
			break
		}
		page++
	}

	utils.Log("count", len(reposArr)).Info("Repos downloading complete")
	return reposArr, nil
}
