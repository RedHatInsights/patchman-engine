package evaluator

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func analyzePackages(tx *gorm.DB, system *models.SystemPlatform, vmaasData *vmaas.UpdatesV3Response) (
	installed, installable, applicable int, err error) {
	if !enablePackageAnalysis {
		return 0, 0, 0, nil
	}

	err = lazySavePackages(tx, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-lazy-pkg-save").Inc()
		return 0, 0, 0, errors.Wrap(err, "lazy package save failed")
	}

	pkgByName, installed, installable, applicable, err := loadPackages(tx, system, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-pkg-data").Inc()
		return 0, 0, 0, errors.Wrap(err, "Unable to load package data")
	}

	err = updateSystemPackages(tx, system, pkgByName)
	if err != nil {
		evaluationCnt.WithLabelValues("error-system-pkgs").Inc()
		return 0, 0, 0, errors.Wrap(err, "Unable to update system packages")
	}
	return installed, installable, applicable, nil
}

// Add unknown EVRAs into the db if needed
func lazySavePackages(tx *gorm.DB, vmaasData *vmaas.UpdatesV3Response) error {
	if !enableLazyPackageSave {
		return nil
	}
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("lazy-package-save"))

	missingPackages := getMissingPackages(tx, vmaasData)
	err := updatePackageDB(&missingPackages)
	if err != nil {
		return errors.Wrap(err, "packages bulk insert failed")
	}
	return nil
}

func getMissingPackage(tx *gorm.DB, nevra string) *models.Package {
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
			tx.Where("name = ?", parsed.Name).First(&pkgName)
			pkg.NameID = pkgName.ID
		}
	}
	return &pkg
}

// Get packages with known name but version missing in db/cache
func getMissingPackages(tx *gorm.DB, vmaasData *vmaas.UpdatesV3Response) models.PackageSlice {
	updates := vmaasData.GetUpdateList()
	packages := make(models.PackageSlice, 0, len(updates))
	for nevra, update := range updates {
		if pkg := getMissingPackage(tx, nevra); pkg != nil {
			packages = append(packages, *pkg)
		}
		for _, pkgUpdate := range update.GetAvailableUpdates() {
			// don't use pkgUpdate.Package since it might be missing epoch, construct it from name and evra
			updateNevra := fmt.Sprintf("%s-%s", pkgUpdate.GetPackageName(), pkgUpdate.GetEVRA())
			if pkg := getMissingPackage(tx, updateNevra); pkg != nil {
				packages = append(packages, *pkg)
			}
		}
	}
	return packages
}

func updatePackageDB(missing *models.PackageSlice) error {
	// autonomous transaction needs to be committed even if evaluation fails
	if len(*missing) > 0 {
		tx := database.Db.Begin()
		defer tx.Commit()
		// tx.Create() also updates packages with their IDs
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(missing).Error
	}
	return nil
}

func updatePackageNameDB(missing *models.PackageName) error {
	// autonomous transaction needs to be committed even if evaluation fails
	if missing != nil {
		tx := database.Db.Begin()
		defer tx.Commit()
		// tx.Create() also updates packages with their IDs
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(missing).Error
	}
	return nil
}

// Find relevant package data based on vmaas results
func loadPackages(tx *gorm.DB, system *models.SystemPlatform,
	vmaasData *vmaas.UpdatesV3Response) (map[string]namedPackage, int, int, int, error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-load"))

	packages, installed, installable, applicable := packagesFromUpdateList(system, vmaasData)
	err := loadSystemNEVRAsFromDB(tx, system, packages)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(err, "loading packages")
	}

	return packages, installed, installable, applicable, nil
}

func packagesFromUpdateList(system *models.SystemPlatform, vmaasData *vmaas.UpdatesV3Response) (
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
				Name:          pkgMeta.Name,
				PackageID:     pkgMeta.ID,
				EVRA:          pkgMeta.Evra,
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
	utils.LogInfo("inventoryID", system.InventoryID, "packages", numUpdates)
	return packages, installed, installable, applicable
}

func loadSystemNEVRAsFromDB(tx *gorm.DB, system *models.SystemPlatform, packages map[string]namedPackage) error {
	rows, err := tx.Table("system_package2 sp2").
		Select("sp2.name_id, pn.name, sp2.package_id, p.evra, sp2.installable_id, sp2.applicable_id").
		Joins("JOIN package p ON p.id = sp2.package_id").
		Joins("JOIN package_name pn on pn.id = sp2.name_id").
		Where("rh_account_id = ? AND system_id = ?", system.RhAccountID, system.ID).
		Rows()
	if err != nil {
		return err
	}
	var columns namedPackage
	numStored := 0
	defer rows.Close()
	for rows.Next() {
		err = tx.ScanRows(rows, &columns)
		if err != nil {
			return err
		}
		numStored++
		nevra := utils.NEVRAStringE(columns.Name, columns.EVRA, true)
		if p, ok := packages[nevra]; ok {
			if latestPkgsChanged(p, columns) {
				p.Change = Update
			} else {
				p.Change = Keep
			}
		} else {
			packages[nevra] = namedPackage{
				NameID:        columns.NameID,
				PackageID:     columns.PackageID,
				EVRA:          columns.EVRA,
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
	return !strings.HasPrefix(nevra, "gpg-pubkey")
}

func latestPkgsChanged(current, stored namedPackage) bool {
	installableEqual := (current.InstallableID == nil && stored.InstallableID == nil) ||
		(current.InstallableID != nil && stored.InstallableID != nil &&
			current.InstallableID == stored.InstallableID)
	applicableEqual := (current.ApplicableID == nil && stored.ApplicableID == nil) ||
		(current.ApplicableID != nil && stored.ApplicableID != nil &&
			current.ApplicableID == stored.ApplicableID)
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
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-store"))

	nPkgs := len(packagesByNEVRA)
	removedPkgIDs := make([]int64, 0, nPkgs)
	updatedPackages := make([]models.SystemPackage, 0, nPkgs)

	for _, pkg := range packagesByNEVRA {
		switch pkg.Change {
		case Remove:
			removedPkgIDs = append(removedPkgIDs, pkg.NameID)
		case Add:
			fallthrough
		case Update:
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

func deleteOldSystemPackages(tx *gorm.DB, system *models.SystemPlatform, pkgIds []int64) error {
	err := tx.Where("rh_account_id = ? ", system.RhAccountID).
		Where("system_id = ?", system.ID).
		Where("package_id in (?)", pkgIds).
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
	Name          string
	PackageID     int64
	EVRA          string
	InstallableID *int64
	ApplicableID  *int64
	Change        ChangeType
}
