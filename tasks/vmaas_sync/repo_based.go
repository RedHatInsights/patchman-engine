package vmaas_sync

import (
	"app/base"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"app/base/vmaas"
	"app/tasks"
	"maps"
	"net/http"
	"slices"
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
	repos, packages, latestRepoChange, err := getUpdatedReposWithPackages(now, lastRepoBaseEval)
	if latestRepoChange == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	inventoryAIDs, err := getRepoBasedInventoryIDs(repos, packages)
	if err != nil {
		return nil, err
	}

	database.UpdateTimestampKVValue(LastEvalRepoBased, *latestRepoChange)

	return inventoryAIDs, nil
}

func getRepoBasedInventoryIDs(repos []string, packages []string) ([]mqueue.EvalData, error) {
	var ids []mqueue.EvalData
	if len(repos) == 0 || len(packages) == 0 {
		return ids, nil
	}

	query := tasks.CancelableDB().Table("system_platform sp").
		Joins("JOIN system_repo sr ON  sp.rh_account_id = sr.rh_account_id AND sp.id = sr.system_id").
		Joins("JOIN repo ON repo.id = sr.repo_id").
		Joins("JOIN rh_account ra ON ra.id = sp.rh_account_id").
		Joins("JOIN system_package2 spkg ON spkg.rh_account_id = sp.rh_account_id AND spkg.system_id = sp.id").
		Joins("JOIN package_name pn ON pn.id = spkg.name_id").
		Select("distinct sp.inventory_id, sp.rh_account_id, ra.org_id").
		Where("repo.name IN (?)", repos).
		Where("pn.name IN (?)", packages).
		Order("sp.rh_account_id")
	if err := query.Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func getUpdatedRepos(syncStart time.Time) ([]string, error) {
	repoMap, _, err := getVmaasRepos(syncStart, nil, false)
	if err != nil {
		return []string{}, err
	}
	return slices.Collect(maps.Keys(repoMap)), nil
}

func getUpdatedReposWithPackages(syncStart time.Time, modifiedSince *string) ([]string, []string, *time.Time, error) {
	repoMap, lastChange, err := getVmaasRepos(syncStart, modifiedSince, true)
	if err != nil {
		return nil, nil, nil, err
	}
	var affectedRepos = make([]string, 0, len(repoMap))
	var affectedPackages []string

	included := make(map[string]bool) // remember packages already in list
	for repoName, packageList := range repoMap {
		if len(packageList) == 0 {
			continue
		}
		affectedRepos = append(affectedRepos, repoName)
		for _, packageName := range packageList {
			if !included[packageName] {
				included[packageName] = true
				affectedPackages = append(affectedPackages, packageName)
			}
		}
	}
	return affectedRepos, affectedPackages, lastChange, nil
}

// nolint: funlen
func getVmaasRepos(syncStart time.Time, modifiedSince *string, thirdParty bool,
) (map[string][]string, *time.Time, error) {
	page := 1
	var repoPackages = make(map[string][]string)
	var latestRepoChange *time.Time
	var nReposRedhat int
	var nReposThirdParty int
	reposSyncStart := time.Now()
	for {
		reposReq := vmaas.ReposRequest{
			Page:           page,
			RepositoryList: []string{".*"},
			PageSize:       tasks.AdvisoryPageSize,
			ThirdParty:     &thirdParty,
			ModifiedSince:  modifiedSince,
			ShowPackages:   true,
		}

		vmaasCallFunc := func() (interface{}, *http.Response, error) {
			vmaasData := vmaas.ReposResponse{}
			resp, err := vmaasClient.Request(&base.Context, http.MethodPost, vmaasReposURL, &reposReq, &vmaasData)
			return &vmaasData, resp, err
		}

		vmaasDataPtr, err := utils.HTTPCallRetry(vmaasCallFunc, tasks.VmaasCallExpRetry, tasks.VmaasCallMaxRetries)
		if err != nil {
			return nil, nil, err
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
			repoPackages[k] = make([]string, 0)
			for _, repo := range contentSet {
				if repo["third_party"] == (interface{})(true) {
					thirdParty = true
				}
				repoPackages[k] = getRepoUpdatedPackages(repo)
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
	return repoPackages, latestRepoChange, nil
}

func getRepoUpdatedPackages(repo map[string]interface{}) []string {
	var repoPackages []string
	if value, ok := repo["updated_package_names"]; ok {
		if updatedPkgs, ok := value.([]interface{}); ok {
			repoPackages = make([]string, 0, len(updatedPkgs))
			for _, p := range updatedPkgs {
				if pkg, ok := p.(string); ok {
					repoPackages = append(repoPackages, pkg)
				}
			}
		}
	}
	return repoPackages
}
