package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	staleOn           = "on"
	staleOff          = "off"
	lastUploadLast1D  = "last1D"
	lastUploadLast7D  = "last7D"
	lastUploadLast30D = "last30D"
	lastUploadAll     = "all"
)

var (
	vmaasCallCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "How many times vmaas was called with which result",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "vmaas_call",
	}, []string{"type"})

	storeAdvisoriesCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "How many advisories were loaded with which result",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "store_advisories",
	}, []string{"type"})

	storePackagesCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "How many packages were loaded with which result",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "store_packages",
	}, []string{"type"})

	updateInterval = time.Second * 20

	systemsCnt = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help:      "How many systems are stored and how up-to-date they are",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "systems",
	}, []string{"opt_out", "last_upload"})

	advisoriesCnt = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help:      "How many advisories are stored of which type",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "advisories",
	}, []string{"type"})

	packageCnt = prometheus.NewGauge(prometheus.GaugeOpts{
		Help:      "How many packages are stored",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "packages",
	})

	packageNameCnt = prometheus.NewGauge(prometheus.GaugeOpts{
		Help:      "How many package names are stored",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "package_names",
	})

	systemAdvisoriesStats = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help:      "Max advisories per system found of which type",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "system_advisories_stats",
	}, []string{"type"})

	syncDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Help:      "How long it took to sync from vmaas service",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "sync_duration_seconds",
	})

	messageSendDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Help:      "How long it took to send message",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "message_send_duration_seconds",
	})

	advisoriesCountMismatch = prometheus.NewCounter(prometheus.CounterOpts{
		Help:      "How many advisories were not synced after incremental sync",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "advisory_incermental_sync_mismatch",
	})

	enableCyndiMetrics = utils.GetBoolEnvOrDefault("ENABLE_CYNDI_METRICS", true)
)

func RunMetrics() {
	prometheus.MustRegister(vmaasCallCnt, storeAdvisoriesCnt, storePackagesCnt,
		systemsCnt, advisoriesCnt, systemAdvisoriesStats, syncDuration, messageSendDuration, packageCnt, packageNameCnt,
		databaseSizeBytesGaugeVec, databaseProcessesGaugeVec, cyndiSystemsCnt, cyndiTagsCnt,
		advisoriesCountMismatch)

	go runAdvancedMetricsUpdating()

	// create web app
	app := gin.New()
	core.InitProbes(app)
	middlewares.Prometheus().Use(app)

	go base.TryExposeOnMetricsPort(app)

	port := utils.Cfg.PublicPort
	err := utils.RunServer(base.Context, app, port)
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}

func runAdvancedMetricsUpdating() {
	defer utils.LogPanics(true)

	utils.Log().Info("started advanced metrics updating")
	for {
		update()
		time.Sleep(updateInterval)
	}
}

func update() {
	updateSystemMetrics()
	updateAdvisoryMetrics()
	updatePackageMetrics()
	updateSystemAdvisoriesStats()
	updateDBMetrics()

	if enableCyndiMetrics {
		updateCyndiData()
	}
}

func updateSystemMetrics() {
	counts, err := getSystemCounts()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update system metrics")
		return
	}

	for labels, count := range counts {
		systemsCnt.WithLabelValues(labels.Stale, labels.LastUpload).Set(float64(count))
	}
}

type systemsCntLabels struct {
	Stale      string
	LastUpload string
}

// nolint: lll
type systemCountColumns struct {
	OnLast1D   int `json:"on_last1d" query:"count(*) filter (where stale = true and last_upload > (now() - interval '1 day'))" gorm:"column:on_last1d"`
	OnLast7D   int `json:"on_last7d" query:"count(*) filter (where stale = true and last_upload > (now() - interval '7 day'))" gorm:"column:on_last7d"`
	OnLast30D  int `json:"on_last30d" query:"count(*) filter (where stale = true and last_upload > (now() - interval '30 day'))" gorm:"column:on_last30d"`
	OnAll      int `json:"on_all" query:"count(*) filter (where stale = true)" gorm:"column:on_all"`
	OffLast1D  int `json:"off_last1d" query:"count(*) filter (where stale = false and last_upload > (now() - interval '1 day'))" gorm:"column:off_last1d"`
	OffLast7D  int `json:"off_last7d" query:"count(*) filter (where stale = false and last_upload > (now() - interval '7 day'))" gorm:"column:off_last7d"`
	OffLast30D int `json:"off_last30d" query:"count(*) filter (where stale = false and last_upload > (now() - interval '30 day'))" gorm:"column:off_last30d"`
	OffAll     int `json:"off_all" query:"count(*) filter (where stale = false)" gorm:"column:off_all"`
}

var querySystemCountColumns = database.MustGetSelect(&systemCountColumns{})

// Load stored systems counts according to "stale" and "last_upload" properties.
// This mad-looking query will read all 8 metrics at once and faster then 8 different queries with different where parts
// Result is loaded into the map {"stale_on:last1D": 12, "stale_off:last1D": 3, ...}.
func getSystemCounts() (map[systemsCntLabels]int, error) {
	var row systemCountColumns
	err := database.Db.Model(&models.SystemPlatform{}).
		Select(querySystemCountColumns).Take(&row).Error
	if err != nil {
		return nil, errors.Wrap(err, "unable to load systems count metrics")
	}
	counts := map[systemsCntLabels]int{
		{staleOn, lastUploadLast1D}:   row.OnLast1D,
		{staleOn, lastUploadLast7D}:   row.OnLast7D,
		{staleOn, lastUploadLast30D}:  row.OnLast30D,
		{staleOn, lastUploadAll}:      row.OnAll,
		{staleOff, lastUploadLast1D}:  row.OffLast1D,
		{staleOff, lastUploadLast7D}:  row.OffLast7D,
		{staleOff, lastUploadLast30D}: row.OffLast30D,
		{staleOff, lastUploadAll}:     row.OffAll,
	}
	return counts, nil
}

func updateAdvisoryMetrics() {
	other, enh, bug, sec, err := getAdvisoryCounts()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update advisory metrics")
	}
	advisoriesCnt.WithLabelValues("other").Set(float64(other))
	advisoriesCnt.WithLabelValues("enhancement").Set(float64(enh))
	advisoriesCnt.WithLabelValues("bugfix").Set(float64(bug))
	advisoriesCnt.WithLabelValues("security").Set(float64(sec))
}

func updatePackageMetrics() {
	nPackages, err := getPackageCounts()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update package metrics")
	}
	packageCnt.Set(float64(nPackages))

	nPackageNames, err := getPackageNameCounts()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update package_name metrics")
	}
	packageNameCnt.Set(float64(nPackageNames))
}

// nolint: lll
type advisoryColumns struct {
	Other       int64 `query:"count(*) filter (where advisory_type_id not in (1,2,3))" gorm:"column:other"`
	Enhancement int64 `query:"count(*) filter (where advisory_type_id = 1)" gorm:"column:enhancement"`
	Bugfix      int64 `query:"count(*) filter (where advisory_type_id = 2)" gorm:"column:bugfix"`
	Security    int64 `query:"count(*) filter (where advisory_type_id = 3)" gorm:"column:security"`
}

var queryAdvisoryColumns = database.MustGetSelect(&advisoryColumns{})

func getAdvisoryCounts() (other, enh, bug, sec int64, err error) {
	var row advisoryColumns
	err = database.Db.Model(&models.AdvisoryMetadata{}).
		Select(queryAdvisoryColumns).Take(&row).Error
	if err != nil {
		return 0, 0, 0, 0, errors.Wrap(err, "unable to get advisory type counts")
	}

	return row.Other, row.Enhancement, row.Bugfix, row.Security, nil
}

func getPackageCounts() (count int64, err error) {
	err = database.Db.Model(&models.Package{}).Count(&count).Error
	if err != nil {
		return 0, errors.Wrap(err, "Unable to get package table items count")
	}
	return count, nil
}

func getPackageNameCounts() (count int64, err error) {
	err = database.Db.Model(&models.PackageName{}).Count(&count).Error
	if err != nil {
		return 0, errors.Wrap(err, "Unable to get package_name table items count")
	}
	return count, nil
}

func updateSystemAdvisoriesStats() {
	stats, err := getSystemAdvisorieStats()
	if err != nil {
		utils.Log("err", err.Error()).Info()
		stats = SystemAdvisoryStats{}
	}
	systemAdvisoriesStats.WithLabelValues("max_all").Set(float64(stats.MaxAll))
	systemAdvisoriesStats.WithLabelValues("max_enh").Set(float64(stats.MaxEnh))
	systemAdvisoriesStats.WithLabelValues("max_bug").Set(float64(stats.MaxBug))
	systemAdvisoriesStats.WithLabelValues("max_sec").Set(float64(stats.MaxSec))
}

type SystemAdvisoryStats struct {
	MaxAll int
	MaxEnh int
	MaxBug int
	MaxSec int
}

// Old query was inserting ORDER BY "system_platform"."max_all" AND max_all
func getSystemAdvisorieStats() (stats SystemAdvisoryStats, err error) {
	err = database.Db.Raw("SELECT MAX(advisory_count_cache) as max_all, " +
		"MAX(advisory_enh_count_cache) as max_enh,MAX(advisory_bug_count_cache) " +
		"as max_bug, MAX(advisory_sec_count_cache) as max_sec FROM " +
		"system_platform ORDER BY max_all LIMIT 1").Scan(&stats).Error
	if err != nil {
		return stats, errors.Wrap(err, "unable to get system advisory stats from db")
	}
	return stats, nil
}
