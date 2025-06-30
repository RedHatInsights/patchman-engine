package config

import (
	"app/base/utils"
)

var (
	// Use in-memory cache for /advisories/:id API
	EnableAdvisoryDetailCache = utils.PodConfig.GetBool("advisory_detail_cache", true)
	// Size of in-memory advisory cache
	AdvisoryDetailCacheSize = utils.PodConfig.GetInt("advisory_detail_cache_size", 100)
	// Load all advisories into cache at startup
	PreLoadCache = utils.PodConfig.GetBool("advisory_detail_cache_preload", true)
	// Use in-memory package cache
	EnabledPackageCache = utils.PodConfig.GetBool("package_cache", true)

	// Allow filtering by cyndi tags
	EnableCyndiTags = utils.PodConfig.GetBool("cyndi_tags", true)
	// Use precomputed system counts for advisories
	DisableCachedCounts = !utils.PodConfig.GetBool("cache_counts", true)
	// Satellite systems can't be assigned to baselines/templates
	EnableSatelliteFunctionality = utils.PodConfig.GetBool("satellite_functionality", true)

	// Send recalc message for systems which have been assigned to a different baseline
	EnableBaselineChangeEval = utils.PodConfig.GetBool("baseline_change_eval", true)
	// Send recalc message for systems which have been assigned to a different template
	EnableTemplateChangeEval = utils.PodConfig.GetBool("template_change_eval", true)
	// Honor rbac permissions (can be disabled for tests)
	EnableRBACCHeck = utils.PodConfig.GetBool("rbac", true)

	// Expose templates API (feature flag)
	EnableTemplates = utils.PodConfig.GetBool("templates_api", true)

	EnableKessel = utils.PodConfig.GetBool("KESSEL_ENABLED", false)
)
