package database

import (
	"app/base/models"
	"app/base/types"
	"app/base/utils"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type join func(*gorm.DB) *gorm.DB
type joinsT []join

func (j joinsT) apply(tx *gorm.DB) *gorm.DB {
	for _, join := range j {
		tx = join(tx)
	}
	return tx
}

func Systems(tx *gorm.DB, accountID int, groups map[string]string, joins ...join) *gorm.DB {
	tx = tx.Table("system_platform sp").Where("sp.rh_account_id = ?", accountID)
	tx = (joinsT)(joins).apply(tx)
	return InventoryHostsJoin(tx, groups)
}

func SystemAdvisories(tx *gorm.DB, accountID int, groups map[string]string, joins ...join) *gorm.DB {
	tx = Systems(tx, accountID, groups).
		Joins("JOIN system_advisories sa on sa.system_id = sp.id AND sa.rh_account_id = ?", accountID)
	return (joinsT)(joins).apply(tx)
}

func SystemPackagesShort(tx *gorm.DB, accountID int, joins ...join) *gorm.DB {
	tx = tx.Table("system_package2 spkg").
		Where("spkg.rh_account_id = ?", accountID)
	return (joinsT)(joins).apply(tx)
}

func SystemPackages(tx *gorm.DB, accountID int, groups map[string]string, joins ...join) *gorm.DB {
	tx = Systems(tx, accountID, groups).
		Joins("JOIN system_package2 spkg on spkg.system_id = sp.id AND spkg.rh_account_id = ?", accountID).
		Joins("JOIN package p on p.id = spkg.package_id").
		Joins("JOIN package_name pn on pn.id = spkg.name_id")
	return (joinsT)(joins).apply(tx)
}

func Packages(tx *gorm.DB, joins ...join) *gorm.DB {
	tx = tx.Table("package p").
		Joins("JOIN package_name pn on p.name_id = pn.id")
	return (joinsT)(joins).apply(tx)
}

func PackageByName(tx *gorm.DB, pkgName string, joins ...join) *gorm.DB {
	tx = Packages(tx).Where("pn.name = ?", pkgName)
	return (joinsT)(joins).apply(tx)
}

func SystemAdvisoriesByInventoryID(tx *gorm.DB, accountID int, groups map[string]string, inventoryID string,
	joins ...join) *gorm.DB {
	tx = SystemAdvisories(tx, accountID, groups).Where("sp.inventory_id = ?::uuid", inventoryID)
	return (joinsT)(joins).apply(tx)
}

func SystemAdvisoriesBySystemID(accountID int, systemID int64) *gorm.DB {
	query := systemAdvisoriesQuery(accountID).Where("sp.id = ?", systemID)
	return query
}

func AdvisoryMetadata(tx *gorm.DB) *gorm.DB {
	tx = tx.Table("advisory_metadata am")
	return JoinAdvisoryType(tx)
}

func systemAdvisoriesQuery(accountID int) *gorm.DB {
	query := DB.Table("system_advisories sa").Select("sa.*").
		Joins("join system_platform sp ON sa.rh_account_id = sp.rh_account_id AND sa.system_id = sp.id").
		Where("sa.rh_account_id = ? AND sp.rh_account_id = ?", accountID, accountID)
	return query
}

func Timestamp2Str(ts *types.Rfc3339TimestampWithZ) *string {
	if ts == nil {
		return nil
	}
	ret := ts.Time().Format(time.RFC3339Nano)
	return &ret
}

func GetTimestampKVValueStr(key string) (*string, error) {
	ts, err := GetTimestampKVValue(key)
	if err != nil {
		return nil, err
	}
	return Timestamp2Str(ts), nil
}

func GetTimestampKVValue(key string) (*types.Rfc3339TimestampWithZ, error) {
	var timestamps []*types.Rfc3339TimestampWithZ
	err := DB.Model(&models.TimestampKV{}).
		Where("name = ?", key).
		Pluck("value", &timestamps).Error
	if err != nil {
		return nil, err
	}

	if len(timestamps) == 0 {
		return nil, nil
	}

	return timestamps[0], nil
}

func UpdateTimestampKVValue(key string, value time.Time) {
	ts := value.Format(time.RFC3339Nano)
	err := UpdateTimestampKVValueStr(key, ts)
	if err != nil {
		utils.LogError("err", err.Error(), "key", key, "Unable to updated timestamp KV value")
	}
}

func UpdateTimestampKVValueStr(key, value string) error {
	err := DB.Exec("INSERT INTO timestamp_kv (name, value) values (?, ?)"+
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
	sql, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	sqldb, _ := DB.DB()
	_, err = sqldb.Exec(string(sql))
	return err
}

// compare the dialect translated errors(like gorm.ErrDuplicatedKey)
func IsPgErrorCode(db *gorm.DB, err error, expectedGormErr error) bool {
	translatedErr := db.Dialector.(*postgres.Dialector).Translate(err)
	return errors.Is(translatedErr, expectedGormErr)
}

func logAndWait(query string) {
	utils.LogInfo(
		"host", utils.CoreCfg.DBHost,
		"port", utils.CoreCfg.DBPort,
		"user", utils.CoreCfg.DBUser,
		"db_name", utils.CoreCfg.DBName,
		"command", query,
		"PostgreSQL is unavailable - sleeping")
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

	latest := 0
	for _, v := range files {
		s := strings.Split(v.Name(), "_")
		i, err := strconv.Atoi(s[0])
		if err != nil {
			panic("Migration file does not start with number")
		}
		if i > latest {
			latest = i
		}
	}
	return latest
}

// Wait for database service
func DBWait(waitForDB string) {
	var query string
	if utils.CoreCfg.DBHost == "UNSET" {
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
			db, _ := DB.DB()
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

func ReadReplicaConfigured() bool {
	return len(utils.CoreCfg.DBReadReplicaHost) > 0 && utils.CoreCfg.DBReadReplicaPort != 0
}

func InventoryHostsJoin(tx *gorm.DB, groups map[string]string) *gorm.DB {
	tx = tx.Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id")
	if _, ok := groups[utils.KeyGrouped]; !ok {
		if _, ok := groups[utils.KeyUngrouped]; ok {
			// show only systems with '[]' group
			return tx.Where("ih.groups = '[]'")
		}
		// return query without WHERE if there are no groups
		return tx
	}

	db := DB.Where("ih.groups @> ANY (?::jsonb[])", groups[utils.KeyGrouped])
	if _, ok := groups[utils.KeyUngrouped]; ok {
		db = db.Or("ih.groups = '[]'")
	}
	return tx.Where(db)
}

// LEFT JOIN templates to sp (system_platform)
func JoinTemplates(tx *gorm.DB) *gorm.DB {
	return tx.Joins("LEFT JOIN template t ON sp.template_id = t.id AND sp.rh_account_id = t.rh_account_id")
}

// JOIN advisory_metadata to sa (system_advisories)
func JoinAdvisoryMetadata(tx *gorm.DB) *gorm.DB {
	return tx.Joins("JOIN advisory_metadata am ON am.id = sa.advisory_id")
}

// JOIN advisory_type to am (advisory_metadata)
func JoinAdvisoryType(tx *gorm.DB) *gorm.DB {
	return tx.Joins("JOIN advisory_type at ON am.advisory_type_id = at.id")
}

func JoinInstallableApplicablePackages(tx *gorm.DB) *gorm.DB {
	return tx.Joins("LEFT JOIN package pi ON pi.id = spkg.installable_id").
		Joins("LEFT JOIN package pa ON pa.id = spkg.applicable_id")
}

// JOIN package description, summary, advisory
func JoinPackageDetails(tx *gorm.DB) *gorm.DB {
	return tx.Joins("JOIN strings descr ON p.description_hash = descr.id").
		Joins("JOIN strings sum ON p.summary_hash = sum.id").
		Joins("LEFT JOIN advisory_metadata am ON p.advisory_id = am.id")
}
