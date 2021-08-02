package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/database"
	"app/base/utils"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
	"time"
)

const (
	allSystemCount        = "allSystem"
	systemsSapSystemCount = "systemsSapSystem"
	systemsWithTagsCount  = "systemsWithTag"
)

var (
	cyndiSystemsCnt = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help:      "How many systems are stored and how up-to-date they are",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "cyndi_systems",
	}, []string{"type"})

	cyndiTagsCnt = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help:      "How many inventory hosts and which tags are there",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "cyndi_tags_count",
	}, []string{"type"})
)

type InventoryHostsStats struct {
	SystemsCount    int64
	SapCount        int64
	SystemsWithTags int64
}

func updateCyndiData() {
	stats, err := getCyndiData()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update cyndi metrics")
		stats = InventoryHostsStats{}
	}
	cyndiTagsCnt.WithLabelValues(allSystemCount).Set(float64(stats.SystemsCount))
	cyndiTagsCnt.WithLabelValues(systemsSapSystemCount).Set(float64(stats.SapCount))
	cyndiTagsCnt.WithLabelValues(systemsWithTagsCount).Set(float64(stats.SystemsWithTags))
}

func getCyndiData() (stats InventoryHostsStats, err error) {
	err = database.Db.Table("inventory.hosts").Count(&stats.SystemsCount).Error
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update cyndi metrics")
		return stats, err
	}

	err = database.Db.Table("inventory.hosts").Where("system_profile -> 'sap_system' = 'true'").
		Count(&stats.SapCount).Error
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update cyndi metrics")
		return stats, err
	}

	err = database.Db.Table("inventory.hosts").Select("tags").Where("jsonb_array_length(tags) > 0").
		Count(&stats.SystemsWithTags).Error
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update cyndi metrics")
		return stats, err
	}

	return stats, nil
}

func updateCyndiSystemMetrics() {
	counts, err := getCyndiCounts(time.Now())
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update cyndi system metrics")
		return
	}

	for labels, count := range counts {
		cyndiSystemsCnt.WithLabelValues(labels).Set(float64(count))
	}
}

func getCyndiCounts(refTime time.Time) (map[string]int, error) {
	lastUploadKV := map[string]int{lastUploadLast1D: 1, lastUploadLast7D: 7, lastUploadLast30D: 30, lastUploadAll: -1}
	counts := map[string]int{}
	for lastUploadK, lastUploadV := range lastUploadKV {
		systemsQuery := database.Db.Table("inventory.hosts").
			Session(&gorm.Session{PrepareStmt: true})
		systemsQueryOptOutLastUpload := updateCyndiQueryLastUpload(systemsQuery, refTime, lastUploadV)
		var nSystems int64
		err := systemsQueryOptOutLastUpload.Count(&nSystems).Error
		if err != nil {
			return nil, errors.Wrap(err, "unable to load systems counts: "+
				fmt.Sprintf("last_upload_before_days: %v", lastUploadV))
		}
		counts[lastUploadK] = int(nSystems)
	}
	return counts, nil
}

func updateCyndiQueryLastUpload(systemsQuery *gorm.DB, refTime time.Time, lastNDays int) *gorm.DB {
	if lastNDays >= 0 {
		return systemsQuery.Where("updated > ?", refTime.AddDate(0, 0, -lastNDays))
	}
	return systemsQuery
}
