package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

var (
	messagesReceivedCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "websocket_msgs",
	}, []string{"type"})

	vmaasCallCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "vmaas_call",
	}, []string{"type"})

	storeAdvisoriesCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "store_advisories",
	}, []string{"type"})

	updateInterval = time.Second * 20

	systemsCnt = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "systems",
	}, []string{"type"})

	advisoriesCnt = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "advisories",
	}, []string{"type"})

	systemAdvisoriesStats = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "system_advisories_stats",
	}, []string{"type"})

	syncDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "sync_duration_seconds",
	})

	messageSendDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "message_send_duration_seconds",
	})
)

func RunMetrics() {
	prometheus.MustRegister(messagesReceivedCnt, vmaasCallCnt, storeAdvisoriesCnt,
		systemsCnt, advisoriesCnt, systemAdvisoriesStats, syncDuration, messageSendDuration)

	go runAdvancedMetricsUpdating()

	// create web app
	app := gin.New()
	middlewares.Prometheus().Use(app)
	err := app.Run(":8083")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}

func runAdvancedMetricsUpdating() {
	utils.Log().Info("started advanced metrics updating")
	for {
		update()
		time.Sleep(updateInterval)
	}
}

func update() {
	updateSystemMetrics()
	updateAdvisoryMetrics()
	updateSystemAdvisoriesStats()
}

func updateSystemMetrics() {
	optOuted, notOptOuted, err := getSystemCounts()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update system metrics")
	}
	systemsCnt.WithLabelValues("opt_out_on").Set(float64(optOuted))
	systemsCnt.WithLabelValues("opt_out_off").Set(float64(notOptOuted))
}

func getSystemCounts() (optOuted, notOptOuted int, err error) {
	systemsQuery := database.Db.Model(&models.SystemPlatform{})
	err = systemsQuery.Where("opt_out = true").Count(&optOuted).Error
	if err != nil {
		return 0, 0, errors.Wrap(err, "unable to get metric opt_outed systems")
	}

	err = systemsQuery.Where("opt_out = false").Count(&notOptOuted).Error
	if err != nil {
		return 0, 0, errors.Wrap(err, "unable to get not opt_outed systems")
	}
	return optOuted, notOptOuted, nil
}

func updateAdvisoryMetrics() {
	unknown, enh, bug, sec, err := getAdvisoryCounts()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update advisory metrics")
	}
	advisoriesCnt.WithLabelValues("unknown").Set(float64(unknown))
	advisoriesCnt.WithLabelValues("enhancement").Set(float64(enh))
	advisoriesCnt.WithLabelValues("bugfix").Set(float64(bug))
	advisoriesCnt.WithLabelValues("security").Set(float64(sec))
}

func getAdvisoryCounts() (unknown, enh, bug, sec int, err error) {
	advisoryQuery := database.Db.Model(&models.AdvisoryMetadata{})
	err = advisoryQuery.Where("advisory_type_id = 0").Count(&unknown).Error
	if err != nil {
		return 0, 0, 0, 0, errors.Wrap(err, "unable to get advisories count - type unknown")
	}

	err = advisoryQuery.Where("advisory_type_id = 1").Count(&enh).Error
	if err != nil {
		return 0, 0, 0, 0, errors.Wrap(err, "unable to get advisories count - type enhancement")
	}

	err = advisoryQuery.Where("advisory_type_id = 2").Count(&bug).Error
	if err != nil {
		return 0, 0, 0, 0, errors.Wrap(err, "unable to get advisories count - type bugfix")
	}

	err = advisoryQuery.Where("advisory_type_id = 3").Count(&sec).Error
	if err != nil {
		return 0, 0, 0, 0, errors.Wrap(err, "unable to get advisories count - type security")
	}
	return unknown, enh, bug, sec, nil
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

func getSystemAdvisorieStats() (stats SystemAdvisoryStats, err error) {
	err = database.Db.Table("system_platform").
		Select("MAX(advisory_count_cache) as max_all, MAX(advisory_enh_count_cache) as max_enh," +
			"MAX(advisory_bug_count_cache) as max_bug, MAX(advisory_sec_count_cache) as max_sec").
		First(&stats).Error
	if err != nil {
		return stats, errors.Wrap(err, "unable to get system advisory stats from db")
	}
	return stats, nil
}
