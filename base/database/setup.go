package database

import (
	"app/base/utils"
	"fmt"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	Db *gorm.DB //nolint:stylecheck
)

// Configure Configure database, PostgreSQL or SQLite connection
func Configure() {
	pgConfig := loadEnvPostgreSQLConfig()
	Db = openPostgreSQL(pgConfig)
	check(Db)
}

// PostgreSQLConfig PostgreSQL database config
type PostgreSQLConfig struct {
	Host        string
	Port        int
	User        string
	Database    string
	Passwd      string
	SSLMode     string
	SSLRootCert string
	Debug       bool

	// Additional params.
	StatementTimeoutMs     int // https://www.postgresql.org/docs/10/runtime-config-client.html
	MaxConnections         int
	MaxIdleConnections     int
	MaxConnectionLifetimeS int
}

func createGormConfig(debug bool) *gorm.Config {
	cfg := gorm.Config{}
	if !debug {
		cfg.Logger = logger.Default.LogMode(logger.Silent) // Allow "Slow SQL" warnings on debug mode only.
	}
	return &cfg
}

// open database connection
func openPostgreSQL(dbConfig *PostgreSQLConfig) *gorm.DB {
	connectString := dataSourceName(dbConfig)
	db, err := gorm.Open(postgres.Open(connectString), createGormConfig(dbConfig.Debug))
	if err != nil {
		panic(err)
	}

	if dbConfig.Debug {
		db = db.Debug()
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	sqlDB.SetMaxOpenConns(dbConfig.MaxConnections)
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(time.Duration(dbConfig.MaxConnectionLifetimeS) * time.Second)
	return db
}

// chcek if database connection works
func check(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	err = sqlDB.Ping()
	if err != nil {
		panic(err)
	}
}

// load database config from environment vars using inserted prefix
func loadEnvPostgreSQLConfig() *PostgreSQLConfig {
	port, err := strconv.Atoi(utils.Getenv("DB_PORT", "FILL"))
	if err != nil {
		panic(err)
	}

	config := PostgreSQLConfig{
		User:        utils.Getenv("DB_USER", "FILL"),
		Host:        utils.Getenv("DB_HOST", "FILL"),
		Port:        port,
		Database:    utils.Getenv("DB_NAME", "FILL"),
		Passwd:      utils.Getenv("DB_PASSWD", "FILL"),
		SSLMode:     utils.Getenv("DB_SSLMODE", "FILL"),
		SSLRootCert: utils.Getenv("DB_SSLROOTCERT", ""),
		Debug:       utils.GetBoolEnvOrDefault("DB_DEBUG", false),

		StatementTimeoutMs:     utils.GetIntEnvOrDefault("DB_STATEMENT_TIMEOUT_MS", 0),
		MaxConnections:         utils.GetIntEnvOrDefault("DB_MAX_CONNECTIONS", 250),
		MaxIdleConnections:     utils.GetIntEnvOrDefault("DB_MAX_IDLE_CONNECTIONS", 50),
		MaxConnectionLifetimeS: utils.GetIntEnvOrDefault("DB_MAX_CONNECTION_LIFETIME_S", 60),
	}
	return &config
}

// create "data source" config string needed for database connection opening
func dataSourceName(dbConfig *PostgreSQLConfig) string {
	dbsource := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=%s statement_timeout=%d",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Database, dbConfig.Passwd, dbConfig.SSLMode,
		dbConfig.StatementTimeoutMs)
	if dbConfig.SSLRootCert != "" {
		dbsource = fmt.Sprintf("%s sslrootcert=%s", dbsource, dbConfig.SSLRootCert)
	}
	return dbsource
}
