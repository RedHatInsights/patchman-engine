package config

import (
	"app/base/api"
	"app/base/utils"
	"crypto/tls"
	"crypto/x509"
	"net/http"

	log "github.com/sirupsen/logrus"
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
	// Honor rbac permissions (can be disabled for tests)
	EnableRBACCHeck = utils.PodConfig.GetBool("rbac", true)

	// Expose baselines API (feature flag)
	EnableBaselines = utils.PodConfig.GetBool("baselines_api", true)
	// Expose templates API (feature flag)
	EnableTemplates = utils.PodConfig.GetBool("templates_api", true)

	// Toggle compression when calling Candlepi API
	CandlepinCallCmp = utils.PodConfig.GetBool("candlepin_call_compression", true)
	// Number of retries on Candlepin API
	CandlepinRetries = utils.PodConfig.GetInt("candlepin_retries", 5)
	// Toggle exponential retries on Candlepin API
	CandlepinExpRetries = utils.PodConfig.GetBool("candlepin_exp_retries", true)
	// Debug flag for API calls
	DebugRequest = log.IsLevelEnabled(log.TraceLevel)
)

func CreateCandlepinClient() api.Client {
	getTLSConfig := func() (*tls.Config, error) {
		var tlsConfig *tls.Config
		if utils.CoreCfg.CandlepinCert != "" && utils.CoreCfg.CandlepinKey != "" {
			clientCert, err := tls.X509KeyPair([]byte(utils.CoreCfg.CandlepinCert), []byte(utils.CoreCfg.CandlepinKey))
			if err != nil {
				return nil, err
			}
			certPool, err := x509.SystemCertPool()
			if err != nil {
				return nil, err
			}
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{clientCert},
				RootCAs:      certPool,
				MinVersion:   tls.VersionTLS12,
			}
			utils.LogInfo("using cert to access candlepin")
		}
		return tlsConfig, nil
	}

	tlsConfig, err := getTLSConfig()
	if err != nil {
		utils.LogError("err", err, "parsing candlepin cert")
	}

	return api.Client{
		HTTPClient: &http.Client{Transport: &http.Transport{
			DisableCompression: !CandlepinCallCmp,
			TLSClientConfig:    tlsConfig,
		}},
		Debug: DebugRequest,
	}
}
