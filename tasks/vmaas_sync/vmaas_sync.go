package vmaas_sync

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

	"time"

	"github.com/pkg/errors"
)

var (
	vmaasClient      *api.Client
	vmaasErratasURL  string
	vmaasPkgListURL  string
	vmaasReposURL    string
	vmaasDBChangeURL string
	evalWriter       mqueue.Writer
)

func Configure() {
	core.ConfigureApp()
	vmaasClient = &api.Client{
		HTTPClient: &http.Client{},
		Debug:      tasks.UseTraceLevel,
	}
	vmaasAddress := utils.FailIfEmpty(utils.CoreCfg.VmaasAddress, "VMAAS_ADDRESS")
	vmaasErratasURL = vmaasAddress + base.VMaaSAPIPrefix + "/errata"
	vmaasPkgListURL = vmaasAddress + base.VMaaSAPIPrefix + "/pkglist"
	vmaasReposURL = vmaasAddress + base.VMaaSAPIPrefix + "/repos"
	vmaasDBChangeURL = vmaasAddress + base.VMaaSAPIPrefix + "/dbchange"
	evalTopic := utils.FailIfEmpty(utils.CoreCfg.EvalTopic, "EVAL_TOPIC")
	evalWriter = mqueue.NewKafkaWriterFromEnv(evalTopic)
}

func runSync() {
	utils.LogInfo("Starting vmaas-sync job")

	var lastModified *types.Rfc3339TimestampWithZ
	if tasks.EnableModifiedSinceSync {
		lastModified = GetLastSync(VmaasExported)
	}
	vmaasExportedTS := VmaasDBExported()
	if isSyncNeeded(lastModified, vmaasExportedTS) {
		err := SyncData(lastModified, vmaasExportedTS)
		if err != nil {
			// This probably means programming error, better to exit with nonzero error code, so the error is noticed
			utils.LogFatal("err", err.Error(), "vmaas data sync failed")
		}

		err = SendReevaluationMessages()
		if err != nil {
			utils.LogError("err", err.Error(), "re-evaluation sending routine failed")
		}
	}
}

func GetLastSync(key string) *types.Rfc3339TimestampWithZ {
	ts, err := database.GetTimestampKVValue(key)
	if err != nil {
		utils.LogInfo("ts", ts, "key", key, "Unable to load last sync timestamp")
		return nil
	}
	return ts
}

func SyncData(lastModifiedTS *types.Rfc3339TimestampWithZ, vmaasExportedTS *types.Rfc3339Timestamp) error {
	utils.LogInfo("Data sync started")
	syncStart := time.Now()
	defer utils.ObserveSecondsSince(syncStart, syncDuration)
	lastFullSyncTS := GetLastSync(LastFullSync)

	lastModified := database.Timestamp2Str(lastModifiedTS)
	if lastFullSyncTS != nil {
		nextFullSync := lastFullSyncTS.Time().Add(time.Duration(tasks.FullSyncCadence) * time.Hour)
		if syncStart.After(nextFullSync) {
			lastModified = nil // set to `nil` to do a full vmaas sync
		}
	}

	if tasks.EnableAdvisoriesSync {
		if err := syncAdvisories(syncStart, lastModified); err != nil {
			return errors.Wrap(err, "Failed to sync advisories")
		}
	}

	if tasks.EnablePackagesSync {
		if err := syncPackages(syncStart, lastModified); err != nil {
			return errors.Wrap(err, "Failed to sync packages")
		}
	}

	if tasks.EnableReposSync {
		if err := syncRepos(syncStart); err != nil {
			return errors.Wrap(err, "Failed to sync repos")
		}
	}

	database.UpdateTimestampKVValue(LastSync, syncStart)
	if lastModified == nil {
		database.UpdateTimestampKVValue(LastFullSync, syncStart)
	}
	if vmaasExportedTS != nil {
		database.UpdateTimestampKVValue(VmaasExported, *vmaasExportedTS.Time())
	}

	// refresh caches
	caches.RefreshAdvisoryCaches()

	utils.LogInfo("Data sync finished successfully")
	return nil
}

func RunVmaasSync() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	Configure()

	runSync()
	if err := Metrics().Add(); err != nil {
		utils.LogInfo("err", err, "Could not push to pushgateway")
	}
}
