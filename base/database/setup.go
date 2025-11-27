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
	DB                        *gorm.DB
	DBReadReplica             *gorm.DB
	OtherAdvisoryTypes        []string
	AdvisoryTypes             map[int]string
	globalPgConfig            *PostgreSQLConfig
	globalPgReadReplicaConfig *PostgreSQLConfig
)

func initDB(db *gorm.DB, pgConfig *PostgreSQLConfig, globalPgConfig *PostgreSQLConfig) *gorm.DB {
	if db == nil || pgConfig != globalPgConfig {
		// create new connection if not already exists
		globalPgConfig = pgConfig
		db = openPostgreSQL(pgConfig)
	}
	check(db)
	return db
}

func InitAdminDB() {
	pgConfig := createPostgreSQLPrimaryAdminConfig()
	DB = initDB(DB, pgConfig, globalPgConfig)
}

func InitUserDB() {
	pgConfig := createPostgreSQLPrimaryUserConfig()
	DB = initDB(DB, pgConfig, globalPgConfig)
	if utils.CoreCfg.DBReadReplicaEnabled && ReadReplicaConfigured() {
		pgConfig := createPostgreSQLReplicaConfig()
		DBReadReplica = initDB(DBReadReplica, pgConfig, globalPgReadReplicaConfig)
	}
}

// Configure Configure database, PostgreSQL or SQLite connection
func Configure() {
	InitUserDB()
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
func createPostgreSQLPrimaryConfig() *PostgreSQLConfig {
	config := PostgreSQLConfig{
		Host:                   utils.CoreCfg.DBHost,
		Port:                   utils.CoreCfg.DBPort,
		Database:               utils.CoreCfg.DBName,
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

func createPostgreSQLReplicaConfig() *PostgreSQLConfig {
	config := createPostgreSQLPrimaryConfig()
	config.User = utils.CoreCfg.DBUser
	config.Passwd = utils.CoreCfg.DBPassword
	config.Host = utils.CoreCfg.DBReadReplicaHost
	config.Port = utils.CoreCfg.DBReadReplicaPort
	return config
}

func createPostgreSQLPrimaryUserConfig() *PostgreSQLConfig {
	config := createPostgreSQLPrimaryConfig()
	config.User = utils.CoreCfg.DBUser
	config.Passwd = utils.CoreCfg.DBPassword
	return config
}

func createPostgreSQLPrimaryAdminConfig() *PostgreSQLConfig {
	config := createPostgreSQLPrimaryConfig()
	config.User = utils.CoreCfg.DBAdminUser
	config.Passwd = utils.CoreCfg.DBAdminPassword
	return config
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
