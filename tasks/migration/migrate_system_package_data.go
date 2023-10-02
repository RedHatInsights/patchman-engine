package migration

import (
	"app/base/core"
	"app/base/models"
	"app/base/utils"
	"app/evaluator"
	"app/tasks"
	"encoding/json"
	"fmt"
	"sync"

	"gorm.io/gorm"
)

var memoryPackageCache *evaluator.PackageCache

func RunSystemPackageDataMigration() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	core.ConfigureApp()
	packageCacheSize := utils.GetIntEnvOrDefault("PACKAGE_CACHE_SIZE", 1000000)
	packageNameCacheSize := utils.GetIntEnvOrDefault("PACKAGE_NAME_CACHE_SIZE", 60000)
	memoryPackageCache = evaluator.NewPackageCache(true, true, packageCacheSize, packageNameCacheSize)
	memoryPackageCache.Load()
	utils.LogInfo("Migrating installable/applicable advisories from system_package to system_package2")
	MigrateSystemPackageData()
}

type AccSys struct {
	RhAccountID int
	SystemID    int64
}

type SystemPackageRecord struct {
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
	var wg sync.WaitGroup
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
		utils.LogInfo("#", i, "partition", part, "Migrating partition")
		accSys := getAccSys(part, i)

		// process at most 4 systems at once
		guard := make(chan struct{}, 4)

		for _, as := range accSys {
			guard <- struct{}{}
			wg.Add(1)
			go func(as AccSys, i int, part string) {
				defer func() {
					<-guard
					wg.Done()
				}()
				updates := getUpdates(as, part, i)
				for _, u := range updates {
					updateData := getUpdateData(u, as, part, i)
					latestApplicable, latestInstallable := getEvraApplicability(updateData)
					applicableID, installableID := getPackageIDs(u, i, latestApplicable, latestInstallable)
					if applicableID != 0 && installableID != 0 {
						// insert ids to system_package2
						err = tasks.WithTx(func(db *gorm.DB) error {
							return db.Table("system_package2").
								Where("installable_id IS NULL AND applicable_id IS NULL").
								Save(models.SystemPackage{
									RhAccountID:   as.RhAccountID,
									SystemID:      as.SystemID,
									PackageID:     u.PackageID,
									NameID:        u.NameID,
									InstallableID: &installableID,
									ApplicableID:  &applicableID,
								}).Error
						})
						if err != nil {
							utils.LogWarn("#", i, "Failed to update system_package2")
						}
					}
				}
			}(as, i, part)
		}
		wg.Wait()
		utils.LogInfo("#", i, "partition", part, "Partition migrated")
	}
}

func getAccSys(part string, i int) []AccSys {
	// get systems from system_package partition
	accSys := make([]AccSys, 0)
	err := tasks.WithReadReplicaTx(func(db *gorm.DB) error {
		return db.Table(part).
			Distinct("rh_account_id, system_id").
			Order("rh_account_id").
			Order("system_id").
			Find(&accSys).Error
	})
	if err != nil {
		utils.LogWarn("#", i, "partition", part, "Failed to load data from partition")
		return accSys
	}

	utils.LogInfo("#", i, "partition", part, "count", len(accSys), "Migrating systems")
	return accSys
}

func getUpdates(as AccSys, part string, i int) []SystemPackageRecord {
	var updates []SystemPackageRecord

	// get update_data from system_package for given system
	err := tasks.WithReadReplicaTx(func(db *gorm.DB) error {
		return db.Table(part).
			Select("name_id, package_id, update_data").
			Where("rh_account_id = ?", as.RhAccountID).
			Where("system_id = ?", as.SystemID).
			Find(&updates).Error
	})
	if err != nil {
		utils.LogWarn("#", i, "partition", part, "rh_account_id", as.RhAccountID, "system_id", as.SystemID,
			"err", err.Error(), "Couldn't get update_data")
	}
	return updates
}

func getUpdateData(u SystemPackageRecord, as AccSys, part string, i int) []UpdateData {
	var updateData []UpdateData
	if err := json.Unmarshal(u.UpdateDataJSON, &updateData); err != nil {
		utils.LogWarn("#", i, "partition", part, "rh_account_id", as.RhAccountID, "system_id", as.SystemID,
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

func getPackageIDs(u SystemPackageRecord, i int, latestApplicable, latestInstallable string) (int64, int64) {
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
			nevraApplicable := fmt.Sprintf("%s-%s", name, latestApplicable)
			applicable, applicableInCache = memoryPackageCache.GetByNevra(nevraApplicable)
			if applicableInCache {
				applicableID = applicable.ID
			}
		}

		if len(latestInstallable) > 0 {
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
