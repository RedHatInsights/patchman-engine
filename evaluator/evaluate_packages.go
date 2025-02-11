package evaluator

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func lazySaveAndLoadPackages(system *models.SystemPlatform, vmaasData *vmaas.UpdatesV3Response) (
	map[string]namedPackage, int, int, int, error) {
	if !enablePackageAnalysis {
		utils.LogInfo("package analysis disabled, skipping lazy saving and loading")
		return nil, 0, 0, 0, nil
	}

	err := lazySavePackages(vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-lazy-pkg-save").Inc()
		return nil, 0, 0, 0, errors.Wrap(err, "lazy package save failed")
	}

	pkgByName, installed, installable, applicable, err := loadPackages(system, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-pkg-data").Inc()
		return nil, 0, 0, 0, errors.Wrap(err, "Unable to load package data")
	}
	return pkgByName, installed, installable, applicable, nil
}

// LazySavePackages adds unknown EVRAs into the db if needed.
func lazySavePackages(vmaasData *vmaas.UpdatesV3Response) error {
	if !enableLazyPackageSave {
		return nil
	}
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("lazy-package-save"))

	missingPackages := getMissingPackages(vmaasData)
	err := updatePackageDB(&missingPackages)
	if err != nil {
		return errors.Wrap(err, "packages bulk insert failed")
	}
	return nil
}

func getMissingPackage(nevra string) *models.Package {
	_, found := memoryPackageCache.GetByNevra(nevra)
	if found {
		// package is already in db/cache, nothing needed
		return nil
	}

	utils.LogTrace("missing nevra", nevra, "getMissingPackages")
	parsed, err := utils.ParseNevra(nevra)
	if err != nil {
		utils.LogWarn("err", err.Error(), "nevra", nevra, "Unable to parse nevra")
		return nil
	}

	latestName, found := memoryPackageCache.GetLatestByName(parsed.Name)
	pkg := models.Package{EVRA: parsed.EVRAString()}
	if found {
		// name is known, create missing package in db/cache
		pkg.NameID = latestName.NameID
		pkg.SummaryHash = &latestName.SummaryHash
		pkg.DescriptionHash = &latestName.DescriptionHash
	} else {
		// name is unknown, insert into package_name
		pkgName := models.PackageName{Name: parsed.Name}
		err := updatePackageNameDB(&pkgName)
		if err != nil {
			utils.LogError("err", err.Error(), "nevra", nevra, "unknown package name insert failed")
		}
		pkg.NameID = pkgName.ID
		if pkg.NameID == 0 {
			// insert conflict, it did not return ID
			// try to get ID from package_name table
			database.DB.Where("name = ?", parsed.Name).First(&pkgName)
			pkg.NameID = pkgName.ID
		}
	}
	return &pkg
}

// GetMissingPackages gets packages with a known name but a version missing in db/cache.
func getMissingPackages(vmaasData *vmaas.UpdatesV3Response) models.PackageSlice {
	updates := vmaasData.GetUpdateList()
	packages := make(models.PackageSlice, 0, len(updates))
	for nevra, update := range updates {
		if pkg := getMissingPackage(nevra); pkg != nil {
			packages = append(packages, *pkg)
		}
		for _, pkgUpdate := range update.GetAvailableUpdates() {
			// don't use pkgUpdate.Package since it might be missing epoch, construct it from name and evra
			updateNevra := fmt.Sprintf("%s-%s", pkgUpdate.GetPackageName(), pkgUpdate.GetEVRA())
			if pkg := getMissingPackage(updateNevra); pkg != nil {
				packages = append(packages, *pkg)
			}
		}
	}
	return packages
}

func updatePackageDB(missing *models.PackageSlice) error {
	// autonomous transaction needs to be committed even if evaluation fails
	if len(*missing) > 0 {
		tx := database.DB.Begin()
		defer tx.Commit()
		// tx.Create() also updates packages with their IDs
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(missing).Error
	}
	return nil
}

func updatePackageNameDB(missing *models.PackageName) error {
	// autonomous transaction needs to be committed even if evaluation fails
	if missing != nil {
		tx := database.DB.Begin()
		defer tx.Commit()
		// tx.Create() also updates packages with their IDs
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(missing).Error
	}
	return nil
}

// LoadPackages finds relevant package data based on vmaas results.
func loadPackages(system *models.SystemPlatform, vmaasData *vmaas.UpdatesV3Response) (
	map[string]namedPackage, int, int, int, error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-load"))

	packages, installed, installable, applicable := packagesFromUpdateList(system.InventoryID, vmaasData)
	err := loadSystemNEVRAsFromDB(system, packages)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(err, "loading packages")
	}

	return packages, installed, installable, applicable, nil
}

func packagesFromUpdateList(inventoryID string, vmaasData *vmaas.UpdatesV3Response) (
	map[string]namedPackage, int, int, int) {
	installed := 0
	installable := 0
	applicable := 0
	updates := vmaasData.GetUpdateList()
	numUpdates := len(updates)
	// allocate also space for removed packages
	packages := make(map[string]namedPackage, numUpdates*2)
	for nevra, pkgUpdate := range updates {
		if !isValidNevra(nevra) {
			continue
		}
		installed++
		availableUpdates := pkgUpdate.GetAvailableUpdates()
		pkgMeta, ok := memoryPackageCache.GetByNevra(nevra)
		// before we used nevra.EVRAString() function which shows only non zero epoch, keep it consistent
		// maybe we need here something like: evra := strings.TrimPrefix(upData.GetEVRA(), "0:")
		if ok {
			installableID, applicableID := latestPackagesFromUpdatesList(availableUpdates)
			packages[nevra] = namedPackage{
				NameID:        pkgMeta.NameID,
				PackageID:     pkgMeta.ID,
				Change:        Add,
				InstallableID: installableID,
				ApplicableID:  applicableID,
			}
			if installableID != nil {
				installable++
			}
			if installableID != nil || applicableID != nil {
				applicable++
			}
		}
	}
	utils.LogInfo("inventoryID", inventoryID, "packages", numUpdates)
	return packages, installed, installable, applicable
}

func loadSystemNEVRAsFromDB(system *models.SystemPlatform, packages map[string]namedPackage) error {
	rows, err := database.DB.Table("system_package2 sp2").
		Select("sp2.name_id, sp2.package_id, sp2.installable_id, sp2.applicable_id").
		Where("rh_account_id = ? AND system_id = ?", system.RhAccountID, system.ID).
		Rows()
	if err != nil {
		return err
	}
	var columns namedPackage
	numStored := 0
	defer rows.Close()
	for rows.Next() {
		err = database.DB.ScanRows(rows, &columns)
		if err != nil {
			return err
		}
		numStored++
		pkgCache, ok := memoryPackageCache.GetByID(columns.PackageID)
		if !ok {
			return fmt.Errorf("package missing in cache, package_id: %d", columns.PackageID)
		}
		nevra := utils.NEVRAStringE(pkgCache.Name, pkgCache.Evra, true)
		if p, ok := packages[nevra]; ok {
			if latestPkgsChanged(p, columns) {
				p.Change = Update
			} else {
				p.Change = Keep
			}
			packages[nevra] = p
		} else {
			packages[nevra] = namedPackage{
				NameID:        columns.NameID,
				PackageID:     columns.PackageID,
				Change:        Remove,
				ApplicableID:  columns.ApplicableID,
				InstallableID: columns.InstallableID,
			}
		}
	}

	utils.LogInfo("inventoryID", system.InventoryID, "already stored", numStored)
	return err
}

func isValidNevra(nevra string) bool {
	// skip "phantom" package
	if strings.HasPrefix(nevra, "gpg-pubkey") {
		return false
	}

	// Check whether we have that NEVRA in DB
	// this may happen when package nevra can't be properly parsed
	// e.g. oet-service-elasticsearch-0:14.0.5-TSIN-5527-1.noarch
	_, ok := memoryPackageCache.GetByNevra(nevra)
	return ok
}

func latestPkgsChanged(current, stored namedPackage) bool {
	installableEqual := (current.InstallableID == nil && stored.InstallableID == nil) ||
		(current.InstallableID != nil && stored.InstallableID != nil &&
			*current.InstallableID == *stored.InstallableID)
	applicableEqual := (current.ApplicableID == nil && stored.ApplicableID == nil) ||
		(current.ApplicableID != nil && stored.ApplicableID != nil &&
			*current.ApplicableID == *stored.ApplicableID)
	return !(installableEqual && applicableEqual)
}

func createSystemPackage(system *models.SystemPlatform, pkg namedPackage) models.SystemPackage {
	systemPackage := models.SystemPackage{
		RhAccountID:   system.RhAccountID,
		SystemID:      system.ID,
		PackageID:     pkg.PackageID,
		NameID:        pkg.NameID,
		InstallableID: pkg.InstallableID,
		ApplicableID:  pkg.ApplicableID,
	}
	return systemPackage
}

func updateSystemPackages(tx *gorm.DB, system *models.SystemPlatform,
	packagesByNEVRA map[string]namedPackage) error {
	if !enablePackageAnalysis {
		utils.LogInfo("package analysis disabled, skipping storing")
		return nil
	}
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-store"))

	nPkgs := len(packagesByNEVRA)
	removedPkgIDs := make([]int64, 0, nPkgs)
	updatedPackages := make([]models.SystemPackage, 0, nPkgs)
	uniqPackageIDs := make(map[int64]string, nPkgs)

	nevras := make([]string, 0, len(packagesByNEVRA))
	for nevra := range packagesByNEVRA {
		nevras = append(nevras, nevra)
	}
	slices.Sort(nevras)

	for _, nevra := range nevras {
		pkg := packagesByNEVRA[nevra]
		switch pkg.Change {
		case Remove:
			removedPkgIDs = append(removedPkgIDs, pkg.PackageID)
		case Add:
			fallthrough
		case Update:
			if nevra2, ok := uniqPackageIDs[pkg.PackageID]; ok {
				utils.LogWarn("nevra1", nevra, "nevra2", nevra2, "packageID", pkg.PackageID, "Duplicate packageID")
				continue
			}
			uniqPackageIDs[pkg.PackageID] = nevra
			systemPackage := createSystemPackage(system, pkg)
			updatedPackages = append(updatedPackages, systemPackage)
		}
	}

	if err := deleteOldSystemPackages(tx, system, removedPkgIDs); err != nil {
		return err
	}

	err := database.UnnestInsert(tx,
		`INSERT INTO system_package2 (rh_account_id, system_id, package_id, name_id, installable_id, applicable_id)
				(select * from unnest($1::int[], $2::bigint[], $3::bigint[], $4::bigint[], $5::bigint[], $6::bigint[]))
		 ON CONFLICT (rh_account_id, system_id, package_id)
		 DO UPDATE SET installable_id = EXCLUDED.installable_id, applicable_id = EXCLUDED.applicable_id`, updatedPackages)
	return errors.Wrap(err,
		"Storing system packages")
}

func latestPackagesFromUpdatesList(updatePkgData []vmaas.UpdatesV3ResponseAvailableUpdates) (*int64, *int64) {
	var (
		latestInstallable, latestApplicable string
		installableID, applicableID         *int64
	)
	for _, upData := range updatePkgData {
		nevra := upData.GetPackage()
		if len(nevra) == 0 {
			// no update
			continue
		}
		switch upData.StatusID {
		case INSTALLABLE:
			latestInstallable = nevra
		case APPLICABLE:
			latestApplicable = nevra
		}
	}
	if len(latestInstallable) > 0 {
		if installableFromCache, ok := memoryPackageCache.GetByNevra(latestInstallable); ok {
			installableID = &installableFromCache.ID
		}
	}
	if len(latestApplicable) > 0 {
		if applicableFromCache, ok := memoryPackageCache.GetByNevra(latestApplicable); ok {
			applicableID = &applicableFromCache.ID
		}
	}
	return installableID, applicableID
}

func deleteOldSystemPackages(tx *gorm.DB, system *models.SystemPlatform, pkgIDs []int64) error {
	err := tx.Where("rh_account_id = ? ", system.RhAccountID).
		Where("system_id = ?", system.ID).
		Where("package_id in (?)", pkgIDs).
		Delete(&models.SystemPackage{}).Error

	return errors.Wrap(err, "Deleting outdated system packages")
}

type ChangeType int8

const (
	None ChangeType = iota
	Add
	Keep
	Update
	Remove
)

type namedPackage struct {
	NameID        int64
	PackageID     int64
	InstallableID *int64
	ApplicableID  *int64
	Change        ChangeType
}
