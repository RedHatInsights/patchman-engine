package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base/utils"
	"app/tasks"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	databaseSizeBytesGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help:      "Current database size and tables sizes in bytes",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "database_size_bytes",
	}, []string{"table"})

	databaseProcessesGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help:      "Database processes per particular use",
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "database_processes",
	}, []string{"usename", "state"})
)

func updateDBMetrics() {
	updateMetrics(getTableSizes(), databaseSizeBytesGaugeVec)
	updateMetrics(getDatabaseSize(), databaseSizeBytesGaugeVec)
	updateMetricsWithState(getDatabaseProcesses(), databaseProcessesGaugeVec)
}

// generic structure to load data from database
type keyValue struct {
	Key   string
	Value float64
	State string // used for processes data only
}

func updateMetrics(items []keyValue, metrics *prometheus.GaugeVec) {
	for _, item := range items {
		metrics.WithLabelValues(item.Key).Set(item.Value)
	}
}

func updateMetricsWithState(items []keyValue, metrics *prometheus.GaugeVec) {
	for _, item := range items {
		metrics.WithLabelValues(item.Key, item.State).Set(item.Value)
	}
}

func getTableSizes() []keyValue {
	var tableSizes []keyValue
	err := tasks.CancelableDB().Raw(`select tablename as key, pg_total_relation_size(quote_ident(tablename)) as value
        from (select * from pg_catalog.pg_tables where schemaname = 'public') t;`).
		Find(&tableSizes).Error
	if err != nil {
		utils.LogError("err", err.Error(), "unable to get database table sizes")
	}
	return tableSizes
}

func getDatabaseSize() []keyValue {
	dbName := utils.Cfg.DBName
	var dbSize []keyValue
	err := tasks.CancelableDB().Raw(
		fmt.Sprintf(`SELECT 'database' as key, pg_database_size('%s') as value;`, dbName)).
		Find(&dbSize).Error
	if err != nil {
		utils.LogError("err", err.Error(), "unable to get database total size")
	}

	return dbSize
}

func getDatabaseProcesses() []keyValue {
	var usenameCounts []keyValue
	err := tasks.CancelableDB().Table("pg_stat_activity").
		Select("COALESCE(usename, '-') as key, COUNT(*) as value, COALESCE(state, '-') state").
		Group("key, state").Find(&usenameCounts).Error
	if err != nil {
		utils.LogError("err", err.Error(), "unable to get processes counts")
	}

	return usenameCounts
}
