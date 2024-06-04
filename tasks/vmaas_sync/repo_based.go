package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"app/base/vmaas"
	"app/tasks"
	"net/http"
	"time"
)

const LastEvalRepoBased = "last_eval_repo_based"
const LastSync = "last_sync"
const LastFullSync = "last_full_sync"
const VmaasExported = "vmaas_exported"

func getCurrentRepoBasedInventoryIDs() ([]mqueue.EvalData, error) {
	lastRepoBaseEval, err := database.GetTimestampKVValueStr(LastEvalRepoBased)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	thirdParty := true
	repoPackages, repoNoPackages, latestRepoChange, err := getUpdatedRepos(now, lastRepoBaseEval, &thirdParty)
	if latestRepoChange == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	inventoryAIDs, err := getRepoBasedInventoryIDs(repoPackages, repoNoPackages)
	if err != nil {
		return nil, err
	}

	database.UpdateTimestampKVValue(LastEvalRepoBased, *latestRepoChange)

	return inventoryAIDs, nil
}

func getRepoBasedInventoryIDs(repoPackages [][]string, repos []string) ([]mqueue.EvalData, error) {
	var ids []mqueue.EvalData
	if len(repoPackages) == 0 && len(repos) == 0 {
		return ids, nil
	}

	// unique repo names from both repoPackages and repos
	uniqRepos := make(map[string]bool)
	for _, r := range repoPackages {
		uniqRepos[r[0]] = true
	}
	for _, r := range repos {
		uniqRepos[r] = true
	}
	uniqRepoList := make([]string, 0, len(uniqRepos))
	for k := range uniqRepos {
		uniqRepoList = append(uniqRepoList, k)
	}

	innerQuery := database.DB.Table("system_repo sr").
		Joins("JOIN repo ON repo.id = sr.repo_id").
		Joins("JOIN system_platform sp ON  sp.rh_account_id = sr.rh_account_id AND sp.id = sr.system_id").
		Joins("JOIN rh_account ra ON ra.id = sp.rh_account_id").
		Select("distinct sp.id, sp.inventory_id, sp.rh_account_id, ra.org_id, repo.name").
		Where("repo.name IN (?)", uniqRepoList)
	query := tasks.CancelableDB().Table("(?) as rb", innerQuery).
		Order("rb.rh_account_id").
		Select("distinct rb.inventory_id, rb.rh_account_id, rb.org_id")
	whereQ := database.DB

	if len(repoPackages) > 0 && len(repoPackages) < tasks.MaxChangedPackages {
		query = query.
			Joins("JOIN system_package2 spkg ON spkg.rh_account_id = rb.rh_account_id AND spkg.system_id = rb.id").
			Joins("JOIN package_name pn ON pn.id = spkg.name_id")
		whereQ = whereQ.Where("(rb.name, pn.name) IN (?)", repoPackages)
	} else {
		whereQ = whereQ.Where("rb.name IN (?)", uniqRepoList)
	}

	if len(repos) > 0 {
		whereQ = whereQ.Or("rb.name IN (?)", repos)
	}

	if err := query.Where(whereQ).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// nolint: funlen
func getUpdatedRepos(syncStart time.Time, modifiedSince *string, thirdParty *bool,
) ([][]string, []string, *time.Time, error) {
	page := 1
	var repoPackages [][]string
	var repoNoPackages []string
	var latestRepoChange *time.Time
	var nReposRedhat int
	var nReposThirdParty int
	reposSyncStart := time.Now()
	for {
		reposReq := vmaas.ReposRequest{
			Page:           page,
			RepositoryList: []string{".*"},
			PageSize:       tasks.AdvisoryPageSize,
			ThirdParty:     thirdParty,
			ModifiedSince:  modifiedSince,
			ShowPackages:   true,
		}

		vmaasCallFunc := func() (interface{}, *http.Response, error) {
			vmaasData := vmaas.ReposResponse{}
			resp, err := vmaasClient.Request(&base.Context, http.MethodPost, vmaasReposURL, &reposReq, &vmaasData)
			return &vmaasData, resp, err
		}

		vmaasDataPtr, err := utils.HTTPCallRetry(base.Context, vmaasCallFunc,
			tasks.VmaasCallExpRetry, tasks.VmaasCallMaxRetries)
		if err != nil {
			return nil, nil, nil, err
		}
		vmaasCallCnt.WithLabelValues("success").Inc()
		repos := vmaasDataPtr.(*vmaas.ReposResponse)
		if repos.Pages < 1 {
			utils.LogInfo("No repos returned from VMaaS")
			break
		}

		if repos.LatestRepoChange == nil {
			break
		}
		if latestRepoChange == nil || latestRepoChange.Before(*repos.LatestRepoChange.Time()) {
			// add 1 second to avoid re-evaluation of the latest repo
			// e.g. vmaas returns `2024-01-05T06:39:53.553807+00:00`
			// 		but patch stores to DB `2024-01-05T06:39:53Z`
			// 		then the next request to /repos is made with "modified_since": "2024-01-05T06:39:53Z"
			// 		which again returns repo modified at 2024-01-05T06:39:53.553807
			t := repos.LatestRepoChange.Time().Add(time.Second)
			latestRepoChange = &t
		}

		utils.LogInfo("page", page, "pages", repos.Pages, "count", len(repos.RepositoryList),
			"sync_duration", utils.SinceStr(syncStart, time.Second),
			"repos_sync_duration", utils.SinceStr(reposSyncStart, time.Second),
			"Downloaded repos")

		for k, contentSet := range repos.RepositoryList {
			thirdParty := false
			packages := make(map[string]bool)
			for _, repo := range contentSet {
				if repo["third_party"] == (interface{})(true) {
					thirdParty = true
				}
				repoPackages = append(repoPackages, getRepoUpdatedPackages(k, repo, packages)...)
			}
			if len(packages) == 0 {
				repoNoPackages = append(repoNoPackages, k)
			}

			if thirdParty {
				nReposThirdParty++
			} else {
				nReposRedhat++
			}
		}

		if page == repos.Pages {
			break
		}
		page++
	}

	utils.LogInfo("redhat", nReposRedhat, "thirdparty", nReposThirdParty, "Repos downloading complete")
	return repoPackages, repoNoPackages, latestRepoChange, nil
}

func getRepoUpdatedPackages(contentSetName string, repo map[string]interface{}, packages map[string]bool) [][]string {
	var repoPackages [][]string
	if value, ok := repo["updated_package_names"]; ok {
		if updatedPkgs, ok := value.([]interface{}); ok {
			for _, p := range updatedPkgs {
				if pkg, ok := p.(string); ok && !packages[pkg] {
					packages[pkg] = true
					repoPackages = append(repoPackages, []string{contentSetName, pkg})
				}
			}
		}
	}
	return repoPackages
}
