package evaluator

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func analyzePackages(tx *gorm.DB, system *models.SystemPlatform, vmaasData *vmaas.UpdatesV2Response) (
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
func lazySavePackages(tx *gorm.DB, vmaasData *vmaas.UpdatesV2Response) error {
	if !enableLazyPackageSave {
		return nil
	}
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("lazy-package-save"))

	missingPackages := getMissingPackages(tx, vmaasData)
	err := updatePackageDB(tx, &missingPackages)
	if err != nil {
		return errors.Wrap(err, "packages bulk insert failed")
	}
	updatePackageCache(missingPackages)
	return nil
}

// Get packages with known name but version missing in db/cache
func getMissingPackages(tx *gorm.DB, vmaasData *vmaas.UpdatesV2Response) models.PackageSlice {
	updates := vmaasData.GetUpdateList()
	packages := make(models.PackageSlice, 0, len(updates))
	for nevra := range updates {
		_, found := memoryPackageCache.GetByNevra(nevra)
		if found {
			// package is already in db/cache, nothing needed
			continue
		}
		utils.Log("missing nevra", nevra).Trace("getMissingPackages")
		parsed, err := utils.ParseNevra(nevra)
		if err != nil {
			utils.Log("err", err.Error(), "nevra", nevra).Warn("Unable to parse nevra")
			continue
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
			err := updatePackageNameDB(tx, &pkgName)
			if err != nil {
				utils.Log("err", err.Error(), "nevra", nevra).Error("unknown package name insert failed")
			}
			pkg.NameID = pkgName.ID
			if pkg.NameID == 0 {
				// insert conflict, it did not return ID
				// try to get ID from package_name table
				tx.Where("name = ?", parsed.Name).First(&pkgName)
				pkg.NameID = pkgName.ID
			}
		}
		packages = append(packages, pkg)
	}
	return packages
}

func updatePackageDB(tx *gorm.DB, missing *models.PackageSlice) error {
	// tx.Create() also updates packages with their IDs
	if len(*missing) > 0 {
		err := tx.Transaction(func(tx2 *gorm.DB) error {
			return tx2.Clauses(clause.OnConflict{DoNothing: true}).Create(missing).Error
		})
		return err
	}
	return nil
}

func updatePackageNameDB(tx *gorm.DB, missing *models.PackageName) error {
	// TODO: use generics once we start using go1.18
	err := tx.Transaction(func(tx2 *gorm.DB) error {
		return tx2.Clauses(clause.OnConflict{DoNothing: true}).Create(missing).Error
	})
	return err
}

func updatePackageCache(missing models.PackageSlice) {
	utils.Log().Trace("updatePackageCache")
	emptyByteSlice := make([]byte, 0)
	for _, dbPkg := range missing {
		name, ok := memoryPackageCache.GetNameByID(dbPkg.NameID)
		if !ok {
			utils.Log("name_id", dbPkg.NameID).Error("name_id missing in memoryPackageCache")
			continue
		}
		descHash := emptyByteSlice
		summaryHash := emptyByteSlice
		if dbPkg.DescriptionHash != nil {
			descHash = *dbPkg.DescriptionHash
		}
		if dbPkg.SummaryHash != nil {
			summaryHash = *dbPkg.SummaryHash
		}
		pkg := PackageCacheMetadata{
			ID:              dbPkg.ID,
			NameID:          dbPkg.NameID,
			Name:            name,
			Evra:            dbPkg.EVRA,
			DescriptionHash: descHash,
			SummaryHash:     summaryHash,
		}
		memoryPackageCache.Add(&pkg)
	}
}

// Find relevant package data based on vmaas results
func loadPackages(tx *gorm.DB, system *models.SystemPlatform,
	vmaasData *vmaas.UpdatesV2Response) (*map[string]namedPackage, error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-load"))

	packages, err := loadSystemNEVRAsFromDB(tx, system, vmaasData)
	if err != nil {
		return nil, errors.Wrap(err, "loading packages")
	}

	pkgByNevra := packages2NevraMap(packages)
	return &pkgByNevra, nil
}

func packages2NevraMap(packages []namedPackage) map[string]namedPackage {
	pkgByNevra := map[string]namedPackage{}
	for _, p := range packages {
		// make sure nevra contains epoch even if epoch==0
		nevra, err := utils.ParseNameEVRA(p.Name, p.EVRA)
		if err != nil {
			utils.Log("package_id", p.PackageID, "name_id", p.NameID, "name", p.Name, "evra", p.EVRA).
				Warn("packages2NevraMap: cannot parse evra")
			continue
		}
		nevraString := nevra.StringE(true)
		pkgByNevra[nevraString] = p
	}
	return pkgByNevra
}

func loadSystemNEVRAsFromDB(tx *gorm.DB, system *models.SystemPlatform,
	vmaasData *vmaas.UpdatesV2Response) ([]namedPackage, error) {
	updates := vmaasData.GetUpdateList()
	numUpdates := len(updates)
	packageIDs := make([]int, 0, numUpdates)
	packages := make([]namedPackage, 0, numUpdates)
	id2index := map[int]int{}
	i := 0
	for nevra := range updates {
		pkgMeta, ok := memoryPackageCache.GetByNevra(nevra)
		if ok {
			packageIDs = append(packageIDs, pkgMeta.ID)
			p := namedPackage{
				NameID:     pkgMeta.NameID,
				Name:       pkgMeta.Name,
				PackageID:  pkgMeta.ID,
				EVRA:       pkgMeta.Evra,
				WasStored:  false,
				UpdateData: nil,
			}
			packages = append(packages, p)
			id2index[pkgMeta.ID] = i
			i++
		}
	}
	rows, err := tx.Table("system_package").
		Select("package_id, update_data").
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
		packages[index].UpdateData = columns.UpdateData
	}
	utils.Log("inventoryID", system.InventoryID, "packages", numUpdates, "already stored", len(packages)).Info()
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
		utils.Log("nevra", nevra).Trace("Unknown package")
		return false
	}
	return true
}

func updateDataChanged(currentNamedPackage *namedPackage, updateDataJSON []byte) bool {
	if bytes.Equal(updateDataJSON, currentNamedPackage.UpdateData) {
		// If the update_data we want to store is null, we skip only if there was a row for this specific
		// system_package already stored.
		if updateDataJSON == nil && currentNamedPackage.WasStored {
			return false
		}

		// If its not null, then the previous check ensured that the old update data matches new one
		if updateDataJSON != nil {
			return false
		}
	}
	return true
}

func createSystemPackage(nevra string,
	updateData vmaas.UpdatesV2ResponseUpdateList,
	system *models.SystemPlatform,
	packagesByNEVRA *map[string]namedPackage) (systemPackagePtr *models.SystemPackage, updatesChanged bool) {
	updateDataJSON, err := vmaasResponse2UpdateDataJSON(&updateData)
	if err != nil {
		utils.Log("nevra", nevra).Error("VMaaS updates response parsing failed")
		return nil, false
	}

	// Skip overwriting entries which have the same data as before
	currentNamedPackage := (*packagesByNEVRA)[nevra]
	if !updateDataChanged(&currentNamedPackage, updateDataJSON) {
		return nil, false
	}

	systemPackage := models.SystemPackage{
		RhAccountID: system.RhAccountID,
		SystemID:    system.ID,
		PackageID:   currentNamedPackage.PackageID,
		UpdateData:  updateDataJSON,
		NameID:      currentNamedPackage.NameID,
	}
	return &systemPackage, true
}

func updateSystemPackages(tx *gorm.DB, system *models.SystemPlatform,
	packagesByNEVRA *map[string]namedPackage,
	vmaasData *vmaas.UpdatesV2Response) (installed, updatable int, err error) {
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
	tx = database.OnConflictUpdateMulti(tx, []string{"rh_account_id", "system_id", "package_id"}, "update_data")
	return installed, updatable, errors.Wrap(database.BulkInsert(tx, toStore), "Storing system packages")
}

func vmaasResponse2UpdateDataJSON(updateData *vmaas.UpdatesV2ResponseUpdateList) ([]byte, error) {
	uniqUpdates := make(map[models.PackageUpdate]bool)
	pkgUpdates := make([]models.PackageUpdate, 0, len(updateData.GetAvailableUpdates()))
	for _, upData := range updateData.GetAvailableUpdates() {
		upNevra, err := utils.ParseNevra(upData.GetPackage())
		// Skip invalid nevras in updates list
		if err != nil {
			utils.Log("nevra", upData.Package).Warn("Invalid nevra")
			continue
		}
		// Keep only unique entries for each update in the list
		pkgUpdate := models.PackageUpdate{
			EVRA: upNevra.EVRAString(), Advisory: upData.GetErratum(),
		}
		if !uniqUpdates[pkgUpdate] {
			pkgUpdates = append(pkgUpdates, pkgUpdate)
			uniqUpdates[pkgUpdate] = true
		}
	}

	if prunePackageLatestOnly && len(pkgUpdates) > 1 {
		pkgUpdates = pkgUpdates[len(pkgUpdates)-1:]
	}

	var updateDataJSON []byte
	var err error
	if len(pkgUpdates) > 0 {
		updateDataJSON, err = json.Marshal(pkgUpdates)
		if err != nil {
			return nil, errors.Wrap(err, "Serializing pkg json")
		}
	}
	return updateDataJSON, nil
}

func deleteOldSystemPackages(tx *gorm.DB, system *models.SystemPlatform,
	packagesByNEVRA *map[string]namedPackage) error {
	pkgIds := make([]int, 0, len(*packagesByNEVRA))
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
	NameID     int
	Name       string
	PackageID  int
	EVRA       string
	WasStored  bool
	UpdateData []byte
}
