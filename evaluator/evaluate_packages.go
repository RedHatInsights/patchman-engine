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
	installed, updatable int, err error) {
	if !enablePackageAnalysis {
		return 0, 0, nil
	}

	err = lazySavePackages(tx, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-lazy-pkg-save").Inc()
		return 0, 0, errors.Wrap(err, "lazy package save failed")
	}

	pkgByName, err := loadPackages(tx, system, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-pkg-data").Inc()
		return 0, 0, errors.Wrap(err, "Unable to load package data")
	}

	installed, updatable, err = updateSystemPackages(tx, system, pkgByName, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-system-pkgs").Inc()
		return 0, 0, errors.Wrap(err, "Unable to update system packages")
	}
	return installed, updatable, nil
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
	vmaasData *vmaas.UpdatesV3Response) (*map[string]namedPackage, error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-load"))

	packages, err := loadSystemNEVRAsFromDB(tx, system, vmaasData)
	if err != nil {
		return nil, errors.Wrap(err, "loading packages")
	}

	pkgByNevra := packages2NevraMap(packages)
	return &pkgByNevra, nil
}

func packages2NevraMap(packages []namedPackage) map[string]namedPackage {
	pkgByNevra := make(map[string]namedPackage, len(packages))
	for _, p := range packages {
		// make sure nevra contains epoch even if epoch==0
		nevraString := utils.NEVRAStringE(p.Name, p.EVRA, true)
		pkgByNevra[nevraString] = p
	}
	return pkgByNevra
}

func loadSystemNEVRAsFromDB(tx *gorm.DB, system *models.SystemPlatform,
	vmaasData *vmaas.UpdatesV3Response) ([]namedPackage, error) {
	updates := vmaasData.GetUpdateList()
	numUpdates := len(updates)
	packageIDs := make([]int64, 0, numUpdates)
	packages := make([]namedPackage, 0, numUpdates)
	id2index := make(map[int64]int, numUpdates)
	i := 0
	for nevra := range updates {
		pkgMeta, ok := memoryPackageCache.GetByNevra(nevra)
		if ok {
			packageIDs = append(packageIDs, pkgMeta.ID)
			p := namedPackage{
				NameID:    pkgMeta.NameID,
				Name:      pkgMeta.Name,
				PackageID: pkgMeta.ID,
				EVRA:      pkgMeta.Evra,
				WasStored: false,
			}
			packages = append(packages, p)
			id2index[pkgMeta.ID] = i
			i++
		}
	}
	rows, err := tx.Table("system_package2").
		Select("package_id, installable_id, applicable_id").
		Where("rh_account_id = ? AND system_id = ?", system.RhAccountID, system.ID).
		Where("package_id in (?)", packageIDs).
		Rows()
	if err != nil {
		return nil, err
	}
	var columns namedPackage
	for rows.Next() {
		err = tx.ScanRows(rows, &columns)
		if err != nil {
			return nil, err
		}
		index := id2index[columns.PackageID]
		packages[index].WasStored = true
		packages[index].InstallableID = columns.InstallableID
		packages[index].ApplicableID = columns.ApplicableID
	}
	utils.LogInfo("inventoryID", system.InventoryID, "packages", numUpdates, "already stored", len(packages))
	return packages, err
}

func isValidNevra(nevra string, packagesByNEVRA *map[string]namedPackage) bool {
	// skip "phantom" package
	if strings.HasPrefix(nevra, "gpg-pubkey") {
		return false
	}

	// Check whether we have that NEVRA in DB
	currentNamedPackage := (*packagesByNEVRA)[nevra]
	if currentNamedPackage.PackageID == 0 {
		utils.LogTrace("nevra", nevra, "Unknown package")
		return false
	}
	return true
}

func latestPkgsChanged(currentNamedPackage *namedPackage, installableID, applicableID int64) bool {
	currentInstallableID, currentApplicableID := int64(0), int64(0)
	if currentNamedPackage.InstallableID != nil {
		currentInstallableID = *currentNamedPackage.InstallableID
	}
	if currentNamedPackage.ApplicableID != nil {
		currentApplicableID = *currentNamedPackage.ApplicableID
	}

	if installableID == currentInstallableID && applicableID == currentApplicableID {
		// If the update_data we want to store is null, we skip only if there was a row for this specific
		// system_package already stored.
		if installableID == 0 && applicableID == 0 && currentNamedPackage.WasStored {
			return false
		}

		// If its not null, then the previous check ensured that the old update data matches new one
		if installableID != 0 || applicableID != 0 {
			return false
		}
	}
	return true
}

func createSystemPackage(nevra string,
	updateData *vmaas.UpdatesV3ResponseUpdateList,
	system *models.SystemPlatform,
	packagesByNEVRA *map[string]namedPackage) (systemPackagePtr *models.SystemPackage, updatesChanged bool) {
	installableID, applicableID := latestPackagesFromVmaasResponse(updateData)

	// Skip overwriting entries which have the same data as before
	currentNamedPackage := (*packagesByNEVRA)[nevra]
	if !latestPkgsChanged(&currentNamedPackage, installableID, applicableID) {
		return nil, false
	}

	systemPackage := models.SystemPackage{
		RhAccountID: system.RhAccountID,
		SystemID:    system.ID,
		PackageID:   currentNamedPackage.PackageID,
		NameID:      currentNamedPackage.NameID,
	}
	if installableID != 0 {
		systemPackage.InstallableID = &installableID
	}
	if applicableID != 0 {
		systemPackage.ApplicableID = &applicableID
	}
	return &systemPackage, true
}

func updateSystemPackages(tx *gorm.DB, system *models.SystemPlatform,
	packagesByNEVRA *map[string]namedPackage,
	vmaasData *vmaas.UpdatesV3Response) (installed, updatable int, err error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-store"))

	updates := vmaasData.GetUpdateList()
	if err := deleteOldSystemPackages(tx, system, packagesByNEVRA); err != nil {
		return 0, 0, err
	}

	toStore := make([]models.SystemPackage, 0, len(updates))
	for nevra, updateData := range updates {
		isValid := isValidNevra(nevra, packagesByNEVRA)
		if !isValid {
			continue
		}
		installed++
		if len(updateData.GetAvailableUpdates()) > 0 {
			updatable++
		}

		systemPackagePtr, updatesChanged := createSystemPackage(nevra, updateData, system, packagesByNEVRA)
		if updatesChanged {
			toStore = append(toStore, *systemPackagePtr)
		}
	}
	return installed, updatable, errors.Wrap(
		database.UnnestInsert(tx,
			"INSERT INTO system_package2 (rh_account_id, system_id, package_id, name_id, installable_id, applicable_id)"+
				" (select * from unnest($1::int[], $2::bigint[], $3::bigint[], $4::bigint[], $5::bigint[], $6::bigint[]))"+
				" ON CONFLICT (rh_account_id, system_id, package_id)"+
				" DO UPDATE SET installable_id = EXCLUDED.installable_id, applicable_id = EXCLUDED.applicable_id", toStore),
		"Storing system packages")
}

func latestPackagesFromVmaasResponse(updateData *vmaas.UpdatesV3ResponseUpdateList) (int64, int64) {
	var (
		latestInstallable, latestApplicable string
		installableID, applicableID         int64
	)
	uniqUpdates := make(map[string]bool)
	for _, upData := range updateData.GetAvailableUpdates() {
		nevra := upData.GetPackage()
		if len(nevra) == 0 {
			// no update
			continue
		}
		// Keep only unique entries for each update in the list
		if !uniqUpdates[nevra] {
			uniqUpdates[nevra] = true
			switch upData.StatusID {
			case INSTALLABLE:
				latestInstallable = nevra
			case APPLICABLE:
				latestApplicable = nevra
			}
		}
	}
	if len(latestInstallable) > 0 {
		if installableFromCache, ok := memoryPackageCache.GetByNevra(latestInstallable); ok {
			installableID = installableFromCache.ID
		}
	}
	if len(latestApplicable) > 0 {
		if applicableFromCache, ok := memoryPackageCache.GetByNevra(latestApplicable); ok {
			applicableID = applicableFromCache.ID
		}
	}
	return installableID, applicableID
}

func deleteOldSystemPackages(tx *gorm.DB, system *models.SystemPlatform,
	packagesByNEVRA *map[string]namedPackage) error {
	pkgIds := make([]int64, 0, len(*packagesByNEVRA))
	for _, pkg := range *packagesByNEVRA {
		pkgIds = append(pkgIds, pkg.PackageID)
	}

	err := tx.Where("rh_account_id = ? ", system.RhAccountID).
		Where("system_id = ?", system.ID).
		Where("package_id not in (?)", pkgIds).
		Delete(&models.SystemPackage{}).Error

	return errors.Wrap(err, "Deleting outdated system packages")
}

type namedPackage struct {
	NameID        int64
	Name          string
	PackageID     int64
	EVRA          string
	WasStored     bool
	InstallableID *int64
	ApplicableID  *int64
}
