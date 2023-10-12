package evaluator

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"encoding/json"
	"fmt"
	"strconv"
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

	pkgByName, installed, updatable, err := loadPackages(tx, system, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-pkg-data").Inc()
		return 0, 0, errors.Wrap(err, "Unable to load package data")
	}

	err = updateSystemPackages(tx, system, pkgByName)
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
	vmaasData *vmaas.UpdatesV3Response) (map[string]namedPackage, int, int, error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-load"))

	packages, installed, updatable, err := loadSystemNEVRAsFromDB(tx, system, vmaasData)
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "loading packages")
	}
	return packages, installed, updatable, nil
}

// nolint: funlen
func loadSystemNEVRAsFromDB(tx *gorm.DB, system *models.SystemPlatform,
	vmaasData *vmaas.UpdatesV3Response) (map[string]namedPackage, int, int, error) {
	installed := 0
	updatable := 0
	updates := vmaasData.GetUpdateList()
	numUpdates := len(updates)
	packages := make(map[string]namedPackage, numUpdates*2)
	for nevra, pkgUpdate := range updates {
		if !isValidNevra(nevra) {
			continue
		}
		installed++
		availableUpdates := pkgUpdate.GetAvailableUpdates()
		if len(availableUpdates) > 0 {
			updatable++
		}
		pkgMeta, ok := memoryPackageCache.GetByNevra(nevra)
		// before we used nevra.EVRAString() function which shows only non zero epoch, keep it consistent
		// maybe we need here something like: evra := strings.TrimPrefix(upData.GetEVRA(), "0:")
		if ok {
			pkgUpdateData := packageUpdateData(pkgMeta.Evra, availableUpdates)
			packages[nevra] = namedPackage{
				NameID:     pkgMeta.NameID,
				Name:       pkgMeta.Name,
				PackageID:  pkgMeta.ID,
				EVRA:       pkgMeta.Evra,
				Change:     Add,
				UpdateData: pkgUpdateData,
			}
		}
	}

	rows, err := tx.Table("(?) as t", database.SystemPackageDataShort(tx, system.RhAccountID)).
		Joins("JOIN package p ON p.id = t.package_id").
		Joins("JOIN package_name pn on pn.id = p.name_id").
		Select("t.package_id, pn.name, p.name_id, p.evra, t.update_data").
		Where("system_id = ?", system.ID).
		Rows()
	if err != nil {
		return nil, 0, 0, err
	}
	for rows.Next() {
		var packageID int64
		var nameID int64
		var name string
		var evra string
		var jsonb []byte
		var updateData models.PackageUpdateData
		err = rows.Scan(&packageID, &name, &nameID, &evra, &jsonb)
		if err != nil {
			return nil, 0, 0, err
		}
		nevra := utils.NEVRAStringE(name, evra, true)
		err = json.Unmarshal(jsonb, &updateData)
		if err != nil {
			return nil, 0, 0, err
		}
		if p, ok := packages[nevra]; ok {
			if isEqual(p.UpdateData, updateData) {
				p.Change = Keep
			} else {
				p.Change = Update
			}
		} else {
			packages[nevra] = namedPackage{
				NameID:     nameID,
				PackageID:  packageID,
				EVRA:       evra,
				Change:     Remove,
				UpdateData: updateData,
			}
		}
	}
	if err := rows.Close(); err != nil {
		return nil, 0, 0, err
	}
	utils.LogInfo("inventoryID", system.InventoryID, "packages", numUpdates, "already stored", len(packages))
	return packages, installed, updatable, err
}

func packageUpdateData(installedEvra string,
	availableUpdates []vmaas.UpdatesV3ResponseAvailableUpdates) models.PackageUpdateData {
	data := models.PackageUpdateData{Installed: installedEvra}
	for _, p := range availableUpdates {
		if p.Package != nil {
			// before we used nevra.EVRAString() function which shows only non zero epoch, keep it consistent
			evra := strings.TrimPrefix(*p.EVRA, "0:")
			switch p.StatusID {
			case APPLICABLE:
				data.Applicable = evra
			case INSTALLABLE:
				data.Installable = evra
			}
		}
	}
	return data
}

func isEqual(a, b models.PackageUpdateData) bool {
	return a.Applicable == b.Applicable && a.Installable == b.Installable && a.Installed == b.Installed
}

func isValidNevra(nevra string) bool {
	// skip "phantom" package
	return !strings.HasPrefix(nevra, "gpg-pubkey")
}

func updateSystemPackages(tx *gorm.DB, system *models.SystemPlatform,
	packagesByNEVRA map[string]namedPackage) error {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-store"))

	// update system_package_data
	if err := updateSystemPackageData(tx, system, packagesByNEVRA); err != nil {
		return err
	}

	// update package_system_data
	err := updatePackageSystemData(tx, system, packagesByNEVRA)
	return err
}

func systemPackageUpdateData(pkgDataMap map[string]namedPackage) models.SystemPackageUpdateData {
	updateData := make(models.SystemPackageUpdateData, len(pkgDataMap))
	for _, pkg := range pkgDataMap {
		if pkg.Change == Remove {
			continue
		}
		updateData[pkg.PackageID] = pkg.UpdateData
	}
	return updateData
}

func updateSystemPackageData(tx *gorm.DB, system *models.SystemPlatform,
	pkgDataMap map[string]namedPackage) error {
	jsonb, err := json.Marshal(systemPackageUpdateData(pkgDataMap))
	if err != nil {
		return err
	}
	row := models.SystemPackageData{RhAccountID: system.RhAccountID, SystemID: system.ID, UpdateData: jsonb}
	if len(pkgDataMap) > 0 {
		return database.OnConflictUpdateMulti(tx, []string{"rh_account_id", "system_id"}, "update_data").Create(row).Error
	}
	return tx.Delete(&models.SystemPackageData{}, system.RhAccountID, system.ID).Error
}

func updatePackageSystemData(tx *gorm.DB, system *models.SystemPlatform, pkgDataMap map[string]namedPackage) error {
	removeNameIDs := make([]int64, 0, len(pkgDataMap))
	tx = tx.Session(&gorm.Session{PrepareStmt: true})
	for _, pkg := range pkgDataMap {
		switch pkg.Change {
		case Remove:
			removeNameIDs = append(removeNameIDs, pkg.NameID)
		case Add:
			fallthrough
		case Update:
			// handle updated packages
			jsonb, err := json.Marshal(models.PackageSystemUpdateData{system.ID: pkg.UpdateData})
			if err != nil {
				return err
			}
			row := models.PackageSystemData{RhAccountID: system.RhAccountID, PackageNameID: pkg.NameID, UpdateData: jsonb}
			err = database.OnConflictDoUpdateExpr(tx, []string{"rh_account_id", "package_name_id"},
				database.UpExpr{Name: "update_data",
					Expr: gorm.Expr("package_system_data.update_data || ?", jsonb)}).
				Create(row).Error
			if err != nil {
				return err
			}
		}
	}

	// handle removed packages
	if len(removeNameIDs) > 0 {
		err := tx.Model(&models.PackageSystemData{}).
			Where("rh_account_id = ? and package_name_id in (?)", system.RhAccountID, removeNameIDs).
			Update("update_data", gorm.Expr("update_data - ?", strconv.FormatInt(system.ID, 10))).Error
		if err != nil {
			return err
		}
		// remove package names with no systems
		return tx.Where("rh_account_id = ? and package_name_id in (?)", system.RhAccountID, removeNameIDs).
			Where("(update_data IS NULL OR update_date == '{}'::jsonb)").
			Delete(&models.PackageSystemData{}).Error
	}
	return nil
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
	NameID     int64
	Name       string
	PackageID  int64
	EVRA       string
	Change     ChangeType
	UpdateData models.PackageUpdateData
}
