package migration

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/evaluator"
	"app/tasks"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var (
	memoryPackageCache   *evaluator.PackageCache
	packageCacheSize     int
	packageNameCacheSize int
	maxGoroutines        int
)

func configure() {
	core.ConfigureApp()
	tasks.HandleContextCancel(tasks.WaitAndExit)
	maxGoroutines = utils.GetIntEnvOrDefault("MAX_GOROUTINES", 4)
	packageCacheSize = utils.GetIntEnvOrDefault("PACKAGE_CACHE_SIZE", 1000000)
	packageNameCacheSize = utils.GetIntEnvOrDefault("PACKAGE_NAME_CACHE_SIZE", 60000)
	memoryPackageCache = evaluator.NewPackageCache(true, true, packageCacheSize, packageNameCacheSize)
	memoryPackageCache.Load()
}

func RunSystemPackageDataMigration() {
	configure()
	utils.LogInfo("Migrating installable/applicable advisories from system_package to system_package2")
	MigrateSystemPackageData()
}

type AccSys struct {
	RhAccountID int
	SystemID    int64
}

type SystemPackageRecord struct {
	SystemID int64
	PackageUpdate
}

type PackageUpdate struct {
	NameID         int64
	PackageID      int64
	UpdateDataJSON json.RawMessage `gorm:"column:update_data"`
}

type UpdateData struct {
	Evra   string `json:"evra" gorm:"-"`
	Status string `json:"status" gorm:"-"`
}

type Package struct {
	ID   int64
	Evra string
}

func MigrateSystemPackageData() {
	var partitions []string

	err := tasks.WithReadReplicaTx(func(db *gorm.DB) error {
		return db.Table("pg_tables").
			Where("tablename ~ '^system_package_[0-9]+$'").
			Pluck("tablename", &partitions).Error
	})
	if err != nil {
		utils.LogError("err", err.Error(), "Couldn't get partitions for system_package")
		return
	}

	for i, part := range partitions {
		processPartition(part, i)
	}
}

func processPartition(part string, i int) {
	utils.LogInfo("#", i, "partition", part, "Migrating partition")

	var wg sync.WaitGroup
	tx := tasks.CancelableDB().Begin()
	defer tx.Rollback()

	accs := getAccounts(i)
	utils.LogInfo("#", i, "len(account)", len(accs), "partition", part, "Migrating accounts")
	// process at most `maxGoroutines` systems at once
	guard := make(chan struct{}, maxGoroutines)

	for _, acc := range accs {
		systemUpdates := getUpdates(acc, part, i)
		utils.LogInfo("#", i, "len(systems)", len(systemUpdates), "partition", part, "account", acc, "Migrating account")
		for system_id, updates := range systemUpdates {
			guard <- struct{}{}
			wg.Add(1)
			go func(acc int, system_id int64, updates []PackageUpdate, i int, part string) {
				defer func() {
					<-guard
					wg.Done()
				}()
				toInsert := make([]models.SystemPackage, 0, 1000)
				for _, u := range updates {
					updateData := getUpdateData(u, acc, system_id, part, i)
					latestApplicable, latestInstallable := getEvraApplicability(updateData)
					applicableID, installableID := getPackageIDs(u, i, latestApplicable, latestInstallable)
					if applicableID != 0 || installableID != 0 {
						// insert ids to system_package2
						sp := models.SystemPackage{
							RhAccountID: acc,
							SystemID:    system_id,
							PackageID:   u.PackageID,
							NameID:      u.NameID,
						}
						if installableID != 0 {
							sp.InstallableID = &installableID
						}
						if applicableID != 0 {
							sp.ApplicableID = &applicableID
						}
						toInsert = append(toInsert, sp)
					}
				}
				if len(toInsert) > 0 {
					err := database.UnnestInsert(tx,
						"INSERT INTO system_package2 (rh_account_id, system_id, package_id, name_id, installable_id, applicable_id)"+
							" (select * from unnest($1::int[], $2::bigint[], $3::bigint[], $4::bigint[], $5::bigint[], $6::bigint[]))"+
							" ON CONFLICT DO NOTHING", toInsert)
					if err != nil {
						utils.LogWarn("#", i, "err", err.Error(), "account", acc, "system", system_id,
							"Failed to insert to system_package2")
					}
				}
			}(acc, system_id, updates, i, part)
		}
	}

	wg.Wait()
	if err := errors.Wrap(tx.Commit().Error, "Commit"); err != nil {
		utils.LogError("#", i, "partition", part, "err", err, "Failed to migrate partition")
	}
	utils.LogInfo("#", i, "partition", part, "Partition migrated")
}

// get account ids in partition from system_platform table
func getAccounts(i int) []int {
	accs := make([]int, 0)
	err := tasks.WithReadReplicaTx(func(db *gorm.DB) error {
		return db.Table(fmt.Sprintf("system_platform_%d", i)).
			Distinct("rh_account_id").
			Find(&accs).Error
	})
	if err != nil {
		utils.LogWarn("#", i, "Failed to load accounts from partition")
		return accs
	}

	utils.LogInfo("#", i, "count", len(accs), "Migrating accounts")
	return accs
}

func getUpdates(account int, part string, i int) map[int64][]PackageUpdate {
	var updates []SystemPackageRecord

	err := tasks.WithReadReplicaTx(func(db *gorm.DB) error {
		return db.Table(part).
			Select("name_id, package_id, system_id, update_data").
			Where("rh_account_id = ?", account).
			Order("system_id").
			Find(&updates).Error
	})
	if err != nil {
		utils.LogWarn("#", i, "partition", part, "rh_account_id", account,
			"err", err.Error(), "Couldn't get update_data")
	}

	nSystemsEst := len(updates) / 1000 //estimation, len(updates) / 1000 packages per system
	res := make(map[int64][]PackageUpdate, nSystemsEst)
	for _, u := range updates {
		if _, has := res[u.SystemID]; !has {
			res[u.SystemID] = []PackageUpdate{}
		}
		res[u.SystemID] = append(res[u.SystemID], u.PackageUpdate)
	}

	return res
}

func getUpdateData(u PackageUpdate, acc int, sys int64, part string, i int) []UpdateData {
	var updateData []UpdateData
	if err := json.Unmarshal(u.UpdateDataJSON, &updateData); err != nil {
		utils.LogWarn("#", i, "partition", part, "rh_account_id", acc, "system_id", sys,
			"update_data", string(u.UpdateDataJSON),
			"err", err.Error(), "Couldn't unmarshal update_data")
	}
	return updateData
}

func getEvraApplicability(udpateData []UpdateData) (string, string) {
	// get latest applicable and installable evra
	var latestInstallable, latestApplicable string
	for i := len(udpateData) - 1; i >= 0; i-- {
		if len(latestInstallable) > 0 && len(latestApplicable) > 0 {
			break
		}
		evra := udpateData[i].Evra
		switch udpateData[i].Status {
		case "Installable":
			if len(latestInstallable) == 0 {
				latestInstallable = evra
			}
			if len(latestApplicable) == 0 {
				latestApplicable = evra
			}
		case "Applicable":
			if len(latestApplicable) == 0 {
				latestApplicable = evra
			}
		}
	}

	return latestApplicable, latestInstallable
}

// nolint: funlen
func getPackageIDs(u PackageUpdate, i int, latestApplicable, latestInstallable string) (int64, int64) {
	// get package_id for latest installable and applicable packages
	if len(latestApplicable) == 0 && len(latestInstallable) == 0 {
		return 0, 0
	}

	var applicableID, installableID int64

	name, ok := memoryPackageCache.GetNameByID(u.NameID)
	if ok {
		var applicable, installable *evaluator.PackageCacheMetadata
		// assume both evras will be found in cache
		applicableInCache := true
		installableInCache := true

		if len(latestApplicable) > 0 {
			if !strings.Contains(latestApplicable, ":") {
				latestApplicable = fmt.Sprintf("0:%s", latestApplicable)
			}
			nevraApplicable := fmt.Sprintf("%s-%s", name, latestApplicable)
			applicable, applicableInCache = memoryPackageCache.GetByNevra(nevraApplicable)
			if applicableInCache {
				applicableID = applicable.ID
			}
		}

		if len(latestInstallable) > 0 {
			if !strings.Contains(latestInstallable, ":") {
				latestInstallable = fmt.Sprintf("0:%s", latestInstallable)
			}
			nevraInstallable := fmt.Sprintf("%s-%s", name, latestInstallable)
			installable, installableInCache = memoryPackageCache.GetByNevra(nevraInstallable)
			if installableInCache {
				installableID = installable.ID
			}
		}

		if applicableInCache && installableInCache {
			// return ids only if both evras are found in cache
			return applicableID, installableID
		}
	}

	var packages []Package
	err := tasks.WithReadReplicaTx(func(db *gorm.DB) error {
		return db.Table("package").
			Select("id, evra").
			Where("evra IN (?)", []string{latestApplicable, latestInstallable}).
			Where("name_id = ?", u.NameID).
			Find(&packages).Error
	})
	if err != nil {
		utils.LogWarn("#", i, "Failed to load packages")
	}

	for _, p := range packages {
		if p.Evra == latestApplicable {
			applicableID = p.ID
		}
		if p.Evra == latestInstallable {
			installableID = p.ID
		}
	}

	return applicableID, installableID
}
