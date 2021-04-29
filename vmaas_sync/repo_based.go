package vmaas_sync //nolint:golint,stylecheck
import (
	"app/base"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"net/http"
	"time"
)

const LastEvalRepoBased = "last_eval_repo_based"
const LastSync = "last_sync"

func getCurrentRepoBasedInventoryIDs() ([]mqueue.InventoryAID, error) {
	lastRepoBaseEval, err := database.GetTimestampKVValueStr(LastEvalRepoBased)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	updateRepos, err := getUpdatedRepos(now, lastRepoBaseEval, true)
	if err != nil {
		return nil, err
	}

	inventoryAIDs, err := getRepoBasedInventoryIDs(updateRepos)
	if err != nil {
		return nil, err
	}

	database.UpdateTimestampKVValue(now, LastEvalRepoBased)

	return inventoryAIDs, nil
}

func getRepoBasedInventoryIDs(repos []string) ([]mqueue.InventoryAID, error) {
	var ids []mqueue.InventoryAID
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

func getUpdatedRepos(syncStart time.Time, modifiedSince *string, thirdParty bool) ([]string, error) {
	page := float32(1)
	var reposArr []string
	reposSyncStart := time.Now()
	for {
		reposReq := vmaas.ReposRequest{
			Page:           utils.PtrFloat32(page),
			RepositoryList: []string{".*"},
			PageSize:       utils.PtrFloat32(float32(advisoryPageSize)),
			ThirdParty:     utils.PtrBool(thirdParty),
			ModifiedSince:  modifiedSince,
		}

		vmaasCallFunc := func() (interface{}, *http.Response, error) {
			vmaasData, resp, err := vmaasClient.DefaultApi.AppReposHandlerPostPost(base.Context).ReposRequest(reposReq).
				Execute()
			return &vmaasData, resp, err
		}

		vmaasDataPtr, err := utils.HTTPCallRetry(base.Context, vmaasCallFunc, vmaasCallExpRetry, vmaasCallMaxRetries)
		if err != nil {
			return nil, err
		}
		vmaasCallCnt.WithLabelValues("success").Inc()
		repos := vmaasDataPtr.(*vmaas.ReposResponse)
		if repos.GetPages() < 1 {
			utils.Log().Debug("No repos returned from VMaaS")
			break
		}

		utils.Log("page", int(page), "pages", int(repos.GetPages()), "count", len(repos.GetRepositoryList()),
			"sync_duration", utils.SinceStr(syncStart), "repos_sync_duration", utils.SinceStr(reposSyncStart)).
			Debug("Downloaded repos")
		for k := range repos.GetRepositoryList() {
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
