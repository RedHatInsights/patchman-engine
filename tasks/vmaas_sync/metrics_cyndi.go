package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/database"
	"app/base/utils"

	"github.com/prometheus/client_golang/prometheus"
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

// nolint: lll
type cyndiMetricColumns struct {
	AllSystems    int64 `query:"count(*)" gorm:"column:all_systems"`
	SapSystems    int64 `query:"count(*) filter (where system_profile -> 'sap_system' = 'true')" gorm:"column:sap_systems"`
	TaggedSystems int64 `query:"count(*) filter (where jsonb_array_length(tags) > 0)" gorm:"column:tagged_systems"`
	Updated1D     int64 `query:"count(*) filter (where updated > (now() - interval '1 day'))" gorm:"column:updated1d"`
	Updated7D     int64 `query:"count(*) filter (where updated > (now() - interval '7 day'))" gorm:"column:updated7d"`
	Updated30D    int64 `query:"count(*) filter (where updated > (now() - interval '30 day'))" gorm:"column:updated30d"`
}

var queryCyndiMetricColumns = database.MustGetSelect(&cyndiMetricColumns{})

func updateCyndiData() {
	tagStats, systemStats, err := getCyndiData()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update cyndi metrics")
	}

	for label, count := range tagStats {
		cyndiTagsCnt.WithLabelValues(label).Set(float64(count))
	}
	for label, count := range systemStats {
		cyndiSystemsCnt.WithLabelValues(label).Set(float64(count))
	}
}

func getCyndiData() (tagStats map[string]int64, systemStats map[string]int64, err error) {
	var row cyndiMetricColumns
	err = database.Db.Table("inventory.hosts").
		Select(queryCyndiMetricColumns).Take(&row).Error
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to update cyndi metrics")
		return tagStats, systemStats, err
	}
	tagStats = map[string]int64{
		allSystemCount:        row.AllSystems,
		systemsSapSystemCount: row.SapSystems,
		systemsWithTagsCount:  row.TaggedSystems,
	}
	systemStats = map[string]int64{
		lastUploadLast1D:  row.Updated1D,
		lastUploadLast7D:  row.Updated7D,
		lastUploadLast30D: row.Updated30D,
		lastUploadAll:     row.AllSystems,
	}
	return tagStats, systemStats, nil
}
