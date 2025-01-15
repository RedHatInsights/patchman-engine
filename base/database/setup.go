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
	DB                 *gorm.DB
	DBReadReplica      *gorm.DB
	DBLogicalReplica   *gorm.DB
	OtherAdvisoryTypes []string
	AdvisoryTypes      map[int]string
	globalPgConfig     *PostgreSQLConfig
	LReplicaPgConfig   *PostgreSQLConfig
)

func InitDB() {
	pgConfig := loadEnvPostgreSQLConfig(false, false)
	if DB != nil && pgConfig == globalPgConfig {
		// reuse connection
		check(DB)
		return
	}
	globalPgConfig = pgConfig
	DB = openPostgreSQL(pgConfig)
	check(DB)
	if utils.CoreCfg.DBReadReplicaEnabled {
		pgConfig := loadEnvPostgreSQLConfig(ReadReplicaConfigured(), false)
		DBReadReplica = openPostgreSQL(pgConfig)
		check(DBReadReplica)
	}

	if utils.CoreCfg.DBLogicalReplicaEnabled {
		LReplicaPgConfig = loadEnvPostgreSQLConfig(false, LogicalReplicaConfigured())
		DBLogicalReplica = openPostgreSQL(LReplicaPgConfig)
		check(DBLogicalReplica)
	}
}

// Configure Configure database, PostgreSQL or SQLite connection
func Configure() {
	InitDB()
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
	connectString := DataSourceName(dbConfig)
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
func loadEnvPostgreSQLConfig(useReadReplica, useLogicalReplica bool) *PostgreSQLConfig {
	host := utils.CoreCfg.DBHost
	port := utils.CoreCfg.DBPort
	if useReadReplica {
		host = utils.CoreCfg.DBReadReplicaHost
		port = utils.CoreCfg.DBReadReplicaPort
	}
	if useLogicalReplica {
		host = utils.CoreCfg.DBLogicalReplicaHost
		port = utils.CoreCfg.DBLogicalReplicaPort
	}
	config := PostgreSQLConfig{
		User:                   utils.CoreCfg.DBUser,
		Host:                   host,
		Port:                   port,
		Database:               utils.CoreCfg.DBName,
		Passwd:                 utils.CoreCfg.DBPassword,
		SSLMode:                utils.CoreCfg.DBSslMode,
		SSLRootCert:            utils.CoreCfg.DBSslRootCert,
		Debug:                  utils.CoreCfg.DBDebug,
		StatementTimeoutMs:     utils.CoreCfg.DBStatementTimeoutMs,
		MaxConnections:         utils.CoreCfg.DBMaxConnections,
		MaxIdleConnections:     utils.CoreCfg.DBMaxIdleConnections,
		MaxConnectionLifetimeS: utils.CoreCfg.DBMaxConnectionLifetimeS,
	}
	return &config
}

// create "data source" config string needed for database connection opening
func DataSourceName(dbConfig *PostgreSQLConfig) string {
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
	err := DB.Table("advisory_type").
		Where("name NOT IN ('enhancement', 'bugfix', 'security')").
		Order("name").
		Pluck("name", &OtherAdvisoryTypes).Error
	utils.LogDebug("other_advisory_types", OtherAdvisoryTypes, "Other advisory types loaded from DB")
	if err != nil {
		panic(err)
	}

	// Load AdvisoryTypes
	var types []models.AdvisoryType

	err = DB.Table("advisory_type").
		Select("id, name").
		Scan(&types).Error
	utils.LogDebug("advisory_types", types, "Advisory types loaded from DB")
	if err != nil {
		panic(err)
	}

	AdvisoryTypes = make(map[int]string)
	for _, at := range types {
		AdvisoryTypes[at.ID] = at.Name
	}
}
