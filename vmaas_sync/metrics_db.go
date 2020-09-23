package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/database"
	"app/base/utils"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"os"
)

var (
	dbTableSizeBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "db_table_size_bytes",
		Help:      "Show current size of particular tables in bytes",
	}, []string{"table"})
)

func updateDBMetrics() {
	updateTableSizes()
	updateDatabaseSize()
}

func updateTableSizes() {
	tableSizes := getTableSizes()
	for _, tableSize := range tableSizes {
		dbTableSizeBytes.WithLabelValues(tableSize.Name).Set(tableSize.Size)
	}
}

func updateDatabaseSize() {
	dbSizes := getDatabaseSize()
	for _, dbSize := range dbSizes {
		dbTableSizeBytes.WithLabelValues(dbSize.Name).Set(dbSize.Size)
	}
}

func getTableSizes() []ItemSize {
	var tableSizes []ItemSize
	err := database.Db.Raw(`select tablename as name, pg_total_relation_size(quote_ident(tablename)) as size
        from (select * from pg_catalog.pg_tables where schemaname = 'public') t;`).
		Find(&tableSizes).Error
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to get database table sizes")
	}
	return tableSizes
}

type ItemSize struct {
	Name string
	Size float64 // table size in bytes
}

func getDatabaseSize() []ItemSize {
	dbName := os.Getenv("DB_NAME")
	var dbSize []ItemSize
	err := database.Db.Raw(
		fmt.Sprintf(`SELECT 'database' as name, pg_database_size('%s') as size;`, dbName)).
		Find(&dbSize).Error
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to get database total size")
	}

	return dbSize
}
