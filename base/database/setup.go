package database

import (
	"fmt"
	"app/base/utils"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"os"
	"strconv"
	"time"
)

var (
	Db    *gorm.DB
)

// configure database, PostgreSQL or SQLite connection
func Configure() {
	if os.Getenv("DB_TYPE") == "postgres" {
		utils.Log().Info("using PostgreSQL database")
		pgConfig := loadEnvPostgreSQLConfig()
		Db = openPostgreSQL(pgConfig)
		check(Db)
	} else {
		// default type is sqlite
		utils.Log().Info("using SQLite database")
		ConfigureSQLite()
	}
}

// PostgreSQL database config
type PostgreSQLConfig struct {
	Host     string
	Port     int
	User     string
	Database string
	Passwd   string

	// Additional params.
	Timeout               string
	ReadTimeout           string
	WriteTimeout          string
	MaxConnections        int
	MaxIdleConnections    int
	MaxConnectionLifetime int // Second
}

// open database connection
func openPostgreSQL(dbConfig *PostgreSQLConfig) *gorm.DB {
	db, err := gorm.Open("postgres", dataSourceName(dbConfig))
	if err != nil {
		panic(err)
	}
	// Nastavime limity dle configu.
	db.DB().SetMaxIdleConns(dbConfig.MaxConnections)
	db.DB().SetMaxIdleConns(dbConfig.MaxIdleConnections)
	db.DB().SetConnMaxLifetime(time.Duration(dbConfig.MaxConnectionLifetime) * time.Second)
	return db
}

// chcek if database connection works
func check(db *gorm.DB) {
	err := db.DB().Ping()
	if err != nil {
		panic(err)
	}
}

// load database config from env. vars using "DB_" prefix
func loadEnvPostgreSQLConfig() *PostgreSQLConfig {
	config := loadEnvMySQLConfig("DB_")
	return config
}

// load database config from environment vars using inserted prefix
func loadEnvMySQLConfig(envprefix string) *PostgreSQLConfig {
	port, err := strconv.Atoi(utils.Getenv(envprefix + "PORT", "FILL"))
	if err != nil {
		panic(err)
	}

	config := PostgreSQLConfig{
		User: utils.Getenv(envprefix + "USER", "FILL"),
		Host: utils.Getenv(envprefix + "HOST", "FILL"),
		Port: port,
		Database: utils.Getenv(envprefix + "NAME", "FILL"),
		Passwd: utils.Getenv(envprefix + "PASSWD", "FILL"),

		Timeout: "60s",
		ReadTimeout: "60s",
		WriteTimeout: "60s",
		MaxConnections: 250,
		MaxIdleConnections: 50,
		MaxConnectionLifetime: 60,
	}
	return &config
}

// create config string from additional timeout params
func timeoutParams(dbConfig *PostgreSQLConfig) string {
	return fmt.Sprintf(
		"?sql_mode=TRADITIONAL&timeout=%s&readTimeout=%s&writeTimeout=%s&parseTime=true",
		dbConfig.Timeout,
		dbConfig.ReadTimeout,
		dbConfig.WriteTimeout,
	)
}

// create "data source" config string needed for database connection opening
func dataSourceName(dbConfig *PostgreSQLConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Database, dbConfig.Passwd)
		// timeoutParams(dbConfig))
}
