package metrics

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

var (
	updateInterval = time.Second * 20

	systemsCnt = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "patchman_engine",
		Subsystem: "manager",
		Name:      "systems",
	}, []string{"type"})

	advisoriesCnt = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "patchman_engine",
		Subsystem: "manager",
		Name:      "advisories",
	}, []string{"type"})
)

func init() {
	prometheus.MustRegister(systemsCnt)
	prometheus.MustRegister(advisoriesCnt)
}

func RunAdvancedMetricsUpdating() {
	utils.Log().Info("started advanced metrics updating")
	for {
		update()
		time.Sleep(updateInterval)
	}
}

func update() {
	updateSystemMetrics()
	updateAdvisoryMetrics()
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
