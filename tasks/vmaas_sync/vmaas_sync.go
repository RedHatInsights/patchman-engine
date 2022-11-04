package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/api"
	"app/base/core"
	"app/base/database"
	"app/base/mqueue"
	"app/base/types"
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
	fullSyncCadence          int
)

func Configure() {
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

	fullSyncCadence = utils.GetIntEnvOrDefault("FULL_SYNC_CADENCE", 24*7) // run full sync once in 7 days by default
}

func runSync() {
	utils.Log().Info("Starting vmaas-sync job")

	var lastModified *types.Rfc3339TimestampWithZ
	if enableModifiedSinceSync {
		lastModified = getLastSync(VmaasExported)
	}
	vmaasExportedTS := VmaasDBExported()
	if isSyncNeeded(lastModified, vmaasExportedTS) {
		err := SyncData(lastModified, vmaasExportedTS)
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

func getLastSync(key string) *types.Rfc3339TimestampWithZ {
	ts, err := database.GetTimestampKVValue(key)
	if err != nil {
		utils.Log("ts", ts, "key", key).Info("Unable to load last sync timestamp")
		return nil
	}
	return ts
}

func SyncData(lastModifiedTS *types.Rfc3339TimestampWithZ, vmaasExportedTS *types.Rfc3339TimestampNoT) error {
	utils.Log().Info("Data sync started")
	syncStart := time.Now()
	defer utils.ObserveSecondsSince(syncStart, syncDuration)
	lastFullSyncTS := getLastSync(LastFullSync)

	lastModified := database.Timestamp2Str(lastModifiedTS)
	if lastFullSyncTS != nil {
		nextFullSync := lastFullSyncTS.Time().Add(time.Duration(fullSyncCadence) * time.Hour)
		if syncStart.After(nextFullSync) {
			lastModified = nil // set to `nil` to do a full vmaas sync
		}
	}

	if enableAdvisoriesSync {
		if err := syncAdvisories(syncStart, lastModified); err != nil {
			return errors.Wrap(err, "Failed to sync advisories")
		}
	}

	if enablePackagesSync {
		if err := syncPackages(syncStart, lastModified); err != nil {
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

	database.UpdateTimestampKVValue(LastSync, syncStart)
	if lastModified == nil {
		database.UpdateTimestampKVValue(LastFullSync, syncStart)
	}
	if vmaasExportedTS != nil {
		database.UpdateTimestampKVValue(VmaasExported, *vmaasExportedTS.Time())
	}
	utils.Log().Info("Data sync finished successfully")
	return nil
}

func RunVmaasSync() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	Configure()

	runSync()
	if err := Metrics().Add(); err != nil {
		utils.Log("err", err).Info("Could not push to pushgateway")
	}
}
