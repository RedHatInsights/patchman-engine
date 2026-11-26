package database

import (
	"app/base"
	"app/base/telemetry"
	"app/base/utils"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBMode int

const (
	User DBMode = iota
	Admin
	Replica
)

var (
	baseDB             *gorm.DB
	baseDBReadReplica  *gorm.DB
	OtherAdvisoryTypes []string
	globalPgConfig     map[DBMode]PostgreSQLConfig = make(map[DBMode]PostgreSQLConfig)
)

func initDB(mode DBMode, db *gorm.DB, pgConfig PostgreSQLConfig) *gorm.DB {
	if db == nil || pgConfig != globalPgConfig[mode] {
		globalPgConfig[mode] = pgConfig
		db = openPostgreSQL(&pgConfig)
	}
	check(db)
	return db
}

func InitDB(mode DBMode) {
	if mode == Replica {
		if utils.CoreCfg.DBReadReplicaEnabled && ReadReplicaConfigured() {
			pgConfig := createPostgreSQLConfig(mode)
			baseDBReadReplica = initDB(mode, baseDBReadReplica, pgConfig)
		}
		return
	}

	pgConfig := createPostgreSQLConfig(mode)
	baseDB = initDB(mode, baseDB, pgConfig)
}

// Configure Configure database, PostgreSQL or SQLite connection
func Configure() {
	InitDB(User)
	InitDB(Replica)
	loadAdditionalParamsFromDB()
}

func ConfigureAdmin() {
	InitDB(Admin)
	loadAdditionalParamsFromDB()
}

// PostgreSQLConfig PostgreSQL database config
type PostgreSQLConfig struct {
	Host        string
	Port        int
	User        string
	Database    string
	Passwd      string // #nosec G117
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
	pgCfg, err := ConnConfig(dbConfig)
	if err != nil {
		panic(err)
	}
	var sqlDB *sql.DB
	if telemetry.SQLEnabled() {
		sqlDB = otelsql.OpenDB(stdlib.GetConnector(*pgCfg),
			otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
			otelsql.WithSpanOptions(telemetry.SQLSpanOptions()))
	} else {
		sqlDB = OpenDB(pgCfg)
	}

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), createGormConfig(dbConfig.Debug))
	if err != nil {
		panic(err)
	}

	if dbConfig.Debug {
		db = db.Debug()
	}
	sqlDB, err = db.DB()
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
func createPostgreSQLConfig(mode DBMode) PostgreSQLConfig {
	config := PostgreSQLConfig{
		User:                   utils.CoreCfg.DBUser,
		Host:                   utils.CoreCfg.DBHost,
		Port:                   utils.CoreCfg.DBPort,
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
	switch mode {
	case User:
		// noop
	case Admin:
		config.User = utils.CoreCfg.DBAdminUser
		config.Passwd = utils.CoreCfg.DBAdminPassword
	case Replica:
		config.Host = utils.CoreCfg.DBReadReplicaHost
		config.Port = utils.CoreCfg.DBReadReplicaPort
	}
	return config
}

// ConnConfig builds a pgx config. Address and TLS are parsed from a keyword string so pgx
// can build TLS fallbacks. User, password, database, and statement_timeout are set on the config.
func ConnConfig(dbConfig *PostgreSQLConfig) (*pgx.ConnConfig, error) {
	dsn := fmt.Sprintf("host=%s port=%d sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.SSLMode)
	if dbConfig.SSLRootCert != "" {
		dsn += " sslrootcert=" + dbConfig.SSLRootCert
	}
	cfg, err := pgx.ParseConfigWithOptions(dsn, pgx.ParseConfigOptions{
		ParseConfigOptions: pgconn.ParseConfigOptions{
			ConnStringAllowedKeys: []string{"host", "port", "sslmode", "sslrootcert"},
		},
	})
	if err != nil {
		return nil, err
	}
	cfg.User = dbConfig.User
	cfg.Password = dbConfig.Passwd
	cfg.Database = dbConfig.Database
	cfg.RuntimeParams["statement_timeout"] = strconv.Itoa(dbConfig.StatementTimeoutMs)
	return cfg, nil
}

// OpenDB opens a database/sql pool. cfg must come from ConnConfig.
func OpenDB(cfg *pgx.ConnConfig) *sql.DB {
	return stdlib.OpenDB(*cfg)
}

func loadAdditionalParamsFromDB() {
	// Load OtherAdvisoryTypes list
	err := DB(base.Context).Table("advisory_type").
		Where("name NOT IN ('enhancement', 'bugfix', 'security')").
		Order("name").
		Pluck("name", &OtherAdvisoryTypes).Error
	utils.LogDebug("other_advisory_types", OtherAdvisoryTypes, "Other advisory types loaded from DB")
	if err != nil {
		panic(err)
	}
}

func DB(ctx context.Context) *gorm.DB {
	if baseDB == nil || ctx == nil {
		return baseDB
	}
	return baseDB.WithContext(ctx)
}

func DBReadReplica(ctx context.Context) *gorm.DB {
	if baseDBReadReplica == nil || ctx == nil {
		return baseDBReadReplica
	}
	return baseDBReadReplica.WithContext(ctx)
}

func TestDB() *gorm.DB {
	return DB(base.Context)
}
