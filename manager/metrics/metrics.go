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
)

func init() {
	prometheus.MustRegister(systemsCnt)
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
