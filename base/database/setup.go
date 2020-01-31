package database

import (
	"app/base/utils"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // postgres used under gorm
	"strconv"
	"time"
)

var (
	Db *gorm.DB
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
	connectString := dataSourceName(dbConfig)
	db, err := gorm.Open("postgres", connectString)
	if err != nil {
		panic(err)
	}

	db.DB().SetMaxOpenConns(dbConfig.MaxConnections)
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

		Timeout:               "60s",
		ReadTimeout:           "60s",
		WriteTimeout:          "60s",
		MaxConnections:        250,
		MaxIdleConnections:    50,
		MaxConnectionLifetime: 60,
	}
	return &config
}

// create "data source" config string needed for database connection opening
func dataSourceName(dbConfig *PostgreSQLConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Database, dbConfig.Passwd)
	// timeoutParams(dbConfig))
}
