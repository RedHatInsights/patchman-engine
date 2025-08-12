package tasks

import (
	"app/base/utils"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	// Skip first N accounts in advisory refresh job, e.g. after failure
	SkipNAccountsRefresh = utils.PodConfig.GetInt("skip_n_accounts_refresh", 0)
	// Remove only LIMIT rows in a run, useful to avoid complete wipe in case of error
	DeleteUnusedDataLimit = utils.PodConfig.GetInt("delete_unused_data_limit", 1000)
	// Remove only LIMIT systems in a run, useful to avoid complete wipe in case of error
	DeleteCulledSystemsLimit = utils.PodConfig.GetInt("delete_culled_systems_limit", 1000)
	// Toggle cyndi metrics reporting
	EnableCyndiMetrics = utils.PodConfig.GetBool("enable_cyndi_metrics", true)
	UseTraceLevel      = log.IsLevelEnabled(log.TraceLevel)
	// Toggle system reevaluation base on changed repos
	EnabledRepoBasedReeval = utils.PodConfig.GetBool("repo_based_re_evaluation", true)
	// Send recalc messages for systems with modified repos
	EnableRecalcMessagesSend = utils.PodConfig.GetBool("recalc_messages_send", true)
	// Toggle advisory sync in vmaas_sync
	EnableAdvisoriesSync = utils.PodConfig.GetBool("advisories_sync", true)
	// Toggle package sync in vmaas_sync
	EnablePackagesSync = utils.PodConfig.GetBool("packages_sync", true)
	// Toggle repo sync in vmaas_sync
	EnableReposSync = utils.PodConfig.GetBool("repos_sync", true)
	// Sync data in vnass_sync based on timestamp
	EnableModifiedSinceSync = utils.PodConfig.GetBool("modified_since_sync", true)
	// Page size for /errata vmass API call
	AdvisoryPageSize = utils.PodConfig.GetInt("errata_page_size", 500)
	// Page size for /packages vmass API call
	PackagesPageSize = utils.PodConfig.GetInt("packages_page_size", 5)
	// Number of retries for vmaas API calls, 0 - retry forever
	VmaasCallMaxRetries = utils.PodConfig.GetInt("vmaas_call_max_retries", 8)
	// Use eponential retry timeouts, false - retry periodically
	VmaasCallExpRetry = utils.PodConfig.GetBool("vmaas_call_exp_retry", true)
	// How ofter run full vmaas sync, 7 days by default
	FullSyncCadence    = utils.PodConfig.GetInt("full_sync_cadence", 24*7)
	MaxChangedPackages = utils.PodConfig.GetInt("max_changed_packages", 30000)
	// prune deleted_system table records older than threshold
	DeletedSystemsThreshold = time.Hour * time.Duration(utils.PodConfig.GetInt("system_delete_hrs", 4))
)
