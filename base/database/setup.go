package database

import (
	"app/base/models"
	"app/base/utils"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	Db                 *gorm.DB //nolint:stylecheck
	OtherAdvisoryTypes []string
	AdvisoryTypes      map[int]string
)

// Configure Configure database, PostgreSQL or SQLite connection
func Configure() {
	pgConfig := loadEnvPostgreSQLConfig()
	Db = openPostgreSQL(pgConfig)
	check(Db)
	loadAdditionalParamsFromDB()
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
	config := PostgreSQLConfig{
		User:                   utils.Cfg.DBUser,
		Host:                   utils.Cfg.DBHost,
		Port:                   utils.Cfg.DBPort,
		Database:               utils.Cfg.DBName,
		Passwd:                 utils.Cfg.DBPassword,
		SSLMode:                utils.Cfg.DBSslMode,
		SSLRootCert:            utils.Cfg.DBSslRootCert,
		Debug:                  utils.Cfg.DBDebug,
		StatementTimeoutMs:     utils.Cfg.DBStatementTimeoutMs,
		MaxConnections:         utils.Cfg.DBMaxConnections,
		MaxIdleConnections:     utils.Cfg.DBMaxIdleConnections,
		MaxConnectionLifetimeS: utils.Cfg.DBMaxConnectionLifetimeS,
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

func loadAdditionalParamsFromDB() {
	// Load OtherAdvisoryTypes list
	err := Db.Table("advisory_type").
		Where("name NOT IN ('enhancement', 'bugfix', 'security')").
		Order("name").
		Pluck("name", &OtherAdvisoryTypes).Error
	utils.Log("other_advisory_types", OtherAdvisoryTypes).Debug("Other advisory types loaded from DB")
	if err != nil {
		panic(err)
	}

	// Load AdvisoryTypes
	var types []models.AdvisoryType

	err = Db.Table("advisory_type").
		Select("id, name").
		Scan(&types).
		Error
	if err != nil {
		panic(err)
	}

	AdvisoryTypes = make(map[int]string)
	for _, at := range types {
		AdvisoryTypes[at.ID] = at.Name
	}
}
