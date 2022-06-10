package database

import (
	"app/base/models"
	"app/base/utils"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgconn"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	PgErrorDuplicateKey = "23505"
)

func Systems(tx *gorm.DB, accountID int) *gorm.DB {
	return tx.Table("system_platform sp").Where("sp.rh_account_id = ?", accountID)
}

func SystemAdvisories(tx *gorm.DB, accountID int) *gorm.DB {
	return Systems(tx, accountID).
		Joins("JOIN system_advisories sa on sa.system_id = sp.id AND sa.rh_account_id = ?", accountID).
		Where("when_patched IS NULL")
}

func SystemPackagesShort(tx *gorm.DB, accountID int) *gorm.DB {
	return tx.Table("system_package spkg").
		Where("spkg.rh_account_id = ?", accountID)
}

func SystemPackages(tx *gorm.DB, accountID int) *gorm.DB {
	return Systems(tx, accountID).
		Joins("JOIN system_package spkg on spkg.system_id = sp.id AND spkg.rh_account_id = ?", accountID).
		Joins("JOIN package p on p.id = spkg.package_id").
		Joins("JOIN package_name pn on pn.id = spkg.name_id")
}

func Packages(tx *gorm.DB) *gorm.DB {
	return tx.Table("package p").
		Joins("JOIN package_name pn on p.name_id = pn.id").
		Joins("JOIN strings descr ON p.description_hash = descr.id").
		Joins("JOIN strings sum ON p.summary_hash = sum.id").
		Joins("LEFT JOIN advisory_metadata am ON p.advisory_id = am.id")
}

func PackageByName(tx *gorm.DB, pkgName string) *gorm.DB {
	return Packages(tx).Where("pn.name = ?", pkgName)
}

func SystemAdvisoriesByInventoryID(tx *gorm.DB, accountID int, inventoryID string) *gorm.DB {
	return SystemAdvisories(tx, accountID).Where("sp.inventory_id = ?::uuid", inventoryID)
}

func SystemAdvisoriesBySystemID(tx *gorm.DB, accountID, systemID int) *gorm.DB {
	query := systemAdvisoriesQuery(tx, accountID).Where("sp.id = ?", systemID)
	return query
}

func systemAdvisoriesQuery(tx *gorm.DB, accountID int) *gorm.DB {
	query := tx.Table("system_advisories sa").Select("sa.*").
		Joins("join system_platform sp ON sa.rh_account_id = sp.rh_account_id AND sa.system_id = sp.id").
		Where("sa.rh_account_id = ? AND sp.rh_account_id = ?", accountID, accountID)
	return query
}

func GetTimestampKVValueStr(key string) (*string, error) {
	var timestrings []*string
	err := Db.Model(&models.TimestampKV{}).
		Where("name = ?", key).
		Pluck("value", &timestrings).Error
	if err != nil {
		return nil, err
	}

	if len(timestrings) == 0 {
		return nil, nil
	}

	return timestrings[0], nil
}

func UpdateTimestampKVValue(value time.Time, key string) {
	ts := value.Format(time.RFC3339)
	err := UpdateTimestampKVValueStr(ts, key)
	if err != nil {
		utils.Log("err", err.Error(), "key", key).Error("Unable to updated timestamp KV value")
	}
}

func UpdateTimestampKVValueStr(value, key string) error {
	err := Db.Exec("INSERT INTO timestamp_kv (name, value) values (?, ?)"+
		"ON CONFLICT (name) DO UPDATE SET value = ?", key, value, value).Error
	return err
}

func PluckInt(tx *gorm.DB, columnName string) int {
	var val int
	err := tx.Pluck(columnName, &val).Error
	if err != nil {
		panic(err)
	}
	return val
}

func ExecFile(filename string) error {
	sql, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	sqldb, _ := Db.DB()
	_, err = sqldb.Exec(string(sql))
	return err
}

func IsPgErrorCode(err error, pgCode string) bool {
	switch e := err.(type) {
	case *pgconn.PgError:
		return e.Code == pgCode
	default:
		return false
	}
}

func logAndWait(query string) {
	utils.Log(
		"host", utils.Cfg.DBHost,
		"port", utils.Cfg.DBPort,
		"user", utils.Cfg.DBUser,
		"db_name", utils.Cfg.DBName,
		"command", query,
	).Info("PostgreSQL is unavailable - sleeping")
	time.Sleep(time.Second)
}

func findLatestMigration() int {
	f, err := os.Open("./database_admin/migrations")
	if err != nil {
		panic("Can't open migration directory")
	}
	files, err := f.Readdir(0)
	if err != nil {
		panic("Can't read migration directory")
	}

	max := 0
	for _, v := range files {
		s := strings.Split(v.Name(), "_")
		i, err := strconv.Atoi(s[0])
		if err != nil {
			panic("Migration file does not start with number")
		}
		if i > max {
			max = i
		}
	}
	return max
}

// Wait for database service
func DBWait(waitForDB string) {
	var query string
	if utils.Cfg.DBHost == "UNSET" {
		log.Info("Skipping PostgreSQL check")
		return
	}
	log.Info("Checking if PostgreSQL is up")
	switch waitForDB {
	case "empty":
		query = ";" // Wait only for empty database.
	case "full":
		// Wait for full schema, all migrations, e.g. before tests (schema_migrations.dirty='f').
		latest := findLatestMigration()
		log.Info("Waiting for schema version ", latest)
		query = fmt.Sprintf("SELECT 1/count(*) FROM schema_migrations WHERE version = %d and dirty='f';", latest)
	default:
		query = "SELECT * FROM schema_migrations;"
	}

	dbDown := true
	for dbDown {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logAndWait(query)
				}
			}()
			db, _ := Db.DB()
			if db != nil {
				if _, err := db.Exec(query); err == nil {
					log.Info("Everything is up - executing command")
					dbDown = false
					return
				}
			}
			logAndWait(query)
		}()
	}
}
