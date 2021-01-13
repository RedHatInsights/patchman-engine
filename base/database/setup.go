package database

import (
	"app/base/utils"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strconv"
	"time"
)

var (
	Db *gorm.DB //nolint:stylecheck
)

// configure database, PostgreSQL or SQLite connection
func Configure() {
	pgConfig := loadEnvPostgreSQLConfig()
	Db = openPostgreSQL(pgConfig)
	check(Db)
}

// PostgreSQL database config
type PostgreSQLConfig struct {
	Host     string
	Port     int
	User     string
	Database string
	Passwd   string
	SSLMode  string

	// Additional params.
	StatementTimeoutMs     int // https://www.postgresql.org/docs/10/runtime-config-client.html
	MaxConnections         int
	MaxIdleConnections     int
	MaxConnectionLifetimeS int
}

// open database connection
func openPostgreSQL(dbConfig *PostgreSQLConfig) *gorm.DB {
	connectString := dataSourceName(dbConfig)
	db, err := gorm.Open(postgres.Open(connectString), &gorm.Config{Logger: logger.Default})
	if err != nil {
		panic(err)
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
		User:     utils.Getenv("DB_USER", "FILL"),
		Host:     utils.Getenv("DB_HOST", "FILL"),
		Port:     port,
		Database: utils.Getenv("DB_NAME", "FILL"),
		Passwd:   utils.Getenv("DB_PASSWD", "FILL"),
		SSLMode:  utils.Getenv("DB_SSLMODE", "FILL"),

		StatementTimeoutMs:     utils.GetIntEnvOrDefault("DB_STATEMENT_TIMEOUT_MS", 0),
		MaxConnections:         utils.GetIntEnvOrDefault("DB_MAX_CONNECTIONS", 250),
		MaxIdleConnections:     utils.GetIntEnvOrDefault("DB_MAX_IDLE_CONNECTIONS", 50),
		MaxConnectionLifetimeS: utils.GetIntEnvOrDefault("DB_MAX_CONNECTION_LIFETIME_S", 60),
	}
	return &config
}

// create "data source" config string needed for database connection opening
func dataSourceName(dbConfig *PostgreSQLConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=%s statement_timeout=%d",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Database, dbConfig.Passwd, dbConfig.SSLMode,
		dbConfig.StatementTimeoutMs)
}
