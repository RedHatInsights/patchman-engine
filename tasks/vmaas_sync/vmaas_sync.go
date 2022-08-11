package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/api"
	"app/base/core"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"app/tasks"
	"app/tasks/caches"
	"net/http"

	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	vmaasClient              *api.Client
	vmaasErratasURL          string
	vmaasPkgListURL          string
	vmaasReposURL            string
	vmaasDBChangeURL         string
	evalWriter               mqueue.Writer
	advisoryPageSize         int
	packagesPageSize         int
	enabledRepoBasedReeval   bool
	enableRecalcMessagesSend bool
	enableAdvisoriesSync     bool
	enablePackagesSync       bool
	enableReposSync          bool
	enableModifiedSinceSync  bool
	vmaasCallExpRetry        bool
	vmaasCallMaxRetries      int
)

func configure() {
	core.ConfigureApp()
	useTraceLevel := strings.ToLower(utils.Getenv("LOG_LEVEL", "INFO")) == "trace"
	vmaasClient = &api.Client{
		HTTPClient: &http.Client{},
		Debug:      useTraceLevel,
	}
	vmaasAddress := utils.FailIfEmpty(utils.Cfg.VmaasAddress, "VMAAS_ADDRESS")
	vmaasErratasURL = vmaasAddress + base.VMaaSAPIPrefix + "/errata"
	vmaasPkgListURL = vmaasAddress + base.VMaaSAPIPrefix + "/pkglist"
	vmaasReposURL = vmaasAddress + base.VMaaSAPIPrefix + "/repos"
	vmaasDBChangeURL = vmaasAddress + base.VMaaSAPIPrefix + "/dbchange"
	evalTopic := utils.FailIfEmpty(utils.Cfg.EvalTopic, "EVAL_TOPIC")
	evalWriter = mqueue.NewKafkaWriterFromEnv(evalTopic)
	enabledRepoBasedReeval = utils.GetBoolEnvOrDefault("ENABLE_REPO_BASED_RE_EVALUATION", true)
	enableRecalcMessagesSend = utils.GetBoolEnvOrDefault("ENABLE_RECALC_MESSAGES_SEND", true)

	enableAdvisoriesSync = utils.GetBoolEnvOrDefault("ENABLE_ADVISORIES_SYNC", true)
	enablePackagesSync = utils.GetBoolEnvOrDefault("ENABLE_PACKAGES_SYNC", true)
	enableReposSync = utils.GetBoolEnvOrDefault("ENABLE_REPOS_SYNC", true)
	enableModifiedSinceSync = utils.GetBoolEnvOrDefault("ENABLE_MODIFIED_SINCE_SYNC", true)

	advisoryPageSize = utils.GetIntEnvOrDefault("ERRATA_PAGE_SIZE", 500)
	packagesPageSize = utils.GetIntEnvOrDefault("PACKAGES_PAGE_SIZE", 5)

	vmaasCallMaxRetries = utils.GetIntEnvOrDefault("VMAAS_CALL_MAX_RETRIES", 0)  // 0 - retry forever
	vmaasCallExpRetry = utils.GetBoolEnvOrDefault("VMAAS_CALL_EXP_RETRY", false) // false - retry periodically
}

func runSync() {
	utils.Log().Info("Starting vmaas-sync job")
	lastSyncTS := getLastSyncIfNeeded()

	if isSyncNeeded(lastSyncTS) {
		err := SyncData(lastSyncTS)
		if err != nil {
			// This probably means programming error, better to exit with nonzero error code, so the error is noticed
			utils.Log("err", err.Error()).Fatal("vmaas data sync failed")
		}

		err = SendReevaluationMessages()
		if err != nil {
			utils.Log("err", err.Error()).Error("re-evaluation sending routine failed")
		}
	}
}

func getLastSyncIfNeeded() *string {
	if !enableModifiedSinceSync {
		return nil
	}

	lastSync, err := database.GetTimestampKVValueStr(LastSync)
	if err != nil {
		utils.Log("err", err).Error("Unable to load last sync timestamp")
		return nil
	}
	return lastSync
}

func SyncData(lastSyncTS *string) error {
	utils.Log().Info("Data sync started")
	syncStart := time.Now()
	defer utils.ObserveSecondsSince(syncStart, syncDuration)

	if enableAdvisoriesSync {
		if err := syncAdvisories(syncStart, lastSyncTS); err != nil {
			return errors.Wrap(err, "Failed to sync advisories")
		}
	}

	if enablePackagesSync {
		if err := syncPackages(syncStart, lastSyncTS); err != nil {
			return errors.Wrap(err, "Failed to sync packages")
		}
	}

	if enableReposSync {
		if err := syncRepos(syncStart); err != nil {
			return errors.Wrap(err, "Failed to sync repos")
		}
	}

	// refresh caches
	caches.RefreshAdvisoryCaches()

	database.UpdateTimestampKVValue(syncStart, LastSync)
	utils.Log().Info("Data sync finished successfully")
	return nil
}

func RunVmaasSync() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	configure()

	go RunMetrics()
	runSync()
}
