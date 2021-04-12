package evaluator

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"bytes"
	"encoding/json"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pkg/errors"
	"strings"
	"time"
)

func analyzePackages(tx *gorm.DB, system *models.SystemPlatform, vmaasData *vmaas.UpdatesV2Response) (
	installed, updatable int, err error) {
	if !enablePackageAnalysis {
		return 0, 0, nil
	}

	names, evras := getNamesAndNevrasLists(vmaasData)
	err = lazySavePackages(tx, names, evras)
	if err != nil {
		evaluationCnt.WithLabelValues("error-lazy-pkg-save").Inc()
		return 0, 0, errors.Wrap(err, "lazy package save failed")
	}

	pkgByName, err := loadPackages(tx, system, names, evras)
	if err != nil {
		evaluationCnt.WithLabelValues("error-pkg-data").Inc()
		return 0, 0, errors.Wrap(err, "Unable to load package data")
	}

	installed, updatable, err = updateSystemPackages(tx, system, pkgByName, vmaasData.GetUpdateList())
	if err != nil {
		evaluationCnt.WithLabelValues("error-system-pkgs").Inc()
		return 0, 0, errors.Wrap(err, "Unable to update system packages")
	}
	return installed, updatable, nil
}

// Add unknown EVRAs into the db if needed
func lazySavePackages(tx *gorm.DB, names, evras []string) error {
	if !enableLazyPackageSave {
		return nil
	}
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("lazy-package-save"))

	packagesMetadata, err := getPackagesMetadata(tx)
	if err != nil {
		return errors.Wrap(err, "package map loading failed")
	}

	toEnsure := getPacakgesToEnsureInDB(names, evras, packagesMetadata)
	txOnConflict := tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	err = database.BulkInsert(txOnConflict, toEnsure)
	if err != nil {
		return errors.Wrap(err, "packages bulk insert failed")
	}
	return nil
}

type PackageMetadata struct {
	NameID          int
	SummaryHash     []byte
	DescriptionHash []byte
}

type packageMetadataModel struct {
	Name string
	PackageMetadata
}

var name2PackageMetadataCache map[string]PackageMetadata

func getPackagesMetadata(tx *gorm.DB) (map[string]PackageMetadata, error) {
	if name2PackageMetadataCache != nil {
		return name2PackageMetadataCache, nil
	}

	rows, err := tx.Table("package").
		Select("DISTINCT ON (name_id) name_id, name, summary_hash, description_hash").
		Joins("JOIN package_name pn ON pn.id = name_id").
		Where("summary_hash IS NOT NULL").
		Where("description_hash IS NOT NULL").
		Order("name_id, evra DESC").Rows()
	if err != nil {
		return nil, errors.Wrap(err, "unable to load package name ID hashes")
	}

	name2PackageMetadataCache = map[string]PackageMetadata{}
	var model packageMetadataModel
	for rows.Next() {
		err = tx.ScanRows(rows, &model)
		if err != nil {
			return nil, errors.Wrap(err, "unable to load package metadata row")
		}
		name2PackageMetadataCache[model.Name] = PackageMetadata{
			NameID:          model.NameID,
			DescriptionHash: model.DescriptionHash,
			SummaryHash:     model.SummaryHash,
		}
	}

	return name2PackageMetadataCache, nil
}

func getPackageMetadata(name string, name2PkgMetadata map[string]PackageMetadata) (*PackageMetadata, bool) {
	if pkgMetadata, ok := name2PkgMetadata[name]; ok {
		return &pkgMetadata, true
	}
	return nil, false
}

// Get packages list with known name ID
func getPacakgesToEnsureInDB(names, evras []string, name2PkgMetadata map[string]PackageMetadata) models.PackageSlice {
	packages := make(models.PackageSlice, 0, len(names))
	for i, name := range names {
		pkgMetadata, found := getPackageMetadata(name, name2PkgMetadata)
		if found {
			pkg := models.Package{
				NameID:          pkgMetadata.NameID,
				EVRA:            evras[i],
				SummaryHash:     &pkgMetadata.SummaryHash,
				DescriptionHash: &pkgMetadata.DescriptionHash,
			}
			packages = append(packages, pkg)
		}
	}
	return packages
}

// Find relevant package data based on vmaas results
func loadPackages(tx *gorm.DB, system *models.SystemPlatform,
	names, evras []string) (*map[utils.Nevra]namedPackage, error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-load"))

	packages, err := loadSystemNEVRAsFromDB(tx, system, names, evras)
	if err != nil {
		return nil, errors.Wrap(err, "loading packages")
	}

	pkgByNevra := packages2NevraMap(packages)
	return &pkgByNevra, nil
}

func packages2NevraMap(packages []namedPackage) map[utils.Nevra]namedPackage {
	pkgByNevra := map[utils.Nevra]namedPackage{}
	for _, p := range packages {
		nevra, err := utils.ParseNameEVRA(p.Name, p.EVRA)
		if err != nil {
			utils.Log("err", err.Error(), "name", p.Name, "evra", p.EVRA).Warn("Unable to parse nevra")
			continue
		}
		pkgByNevra[*nevra] = p
	}
	return pkgByNevra
}

func loadSystemNEVRAsFromDB(tx *gorm.DB, system *models.SystemPlatform, names []string,
	evras []string) ([]namedPackage, error) {
	// Might return more data than we need (one EVRA being applicable to more packages)
	// But it was only way to get somewhat fast query plan which only uses index scans
	var packages []namedPackage
	err := tx.Table("package").
		// We need to have data about the package, and what data we had stored in relation to this system.
		Select("pn.id as name_id, pn.name, package.id as package_id, package.evra,"+
			"(sp.system_id IS NOT NULL) as was_stored, sp.update_data").
		Joins("join package_name pn on package.name_id = pn.id").
		// We need to perform left join, so thats why the parameters are here
		Joins(`left join system_package sp on sp.package_id = package.id AND `+
			`sp.rh_account_id = ? AND sp.system_id = ?`, system.RhAccountID, system.ID).
		Where("pn.name in (?)", names).
		Where("package.evra in (?)", evras).Find(&packages).Error
	return packages, err
}

func getNamesAndNevrasLists(vmaasData *vmaas.UpdatesV2Response) (names, evras []string) {
	names = make([]string, 0, len(vmaasData.GetUpdateList()))
	evras = make([]string, 0, len(vmaasData.GetUpdateList()))
	for nevra := range vmaasData.GetUpdateList() {
		if strings.HasPrefix(nevra, "gpg-pubkey") { // skip "phantom" package
			continue
		}

		// Parse and reformat nevras to avoid issues with 0 epoch
		parsed, err := utils.ParseNevra(nevra)
		if err != nil {
			utils.Log("err", err.Error(), "nevra", nevra).Warn("Unable to parse nevra")
			continue
		}
		names = append(names, parsed.Name)
		evras = append(evras, parsed.EVRAString())
	}
	return names, evras
}

func isValidNevra(nevraStr string, packagesByNEVRA *map[utils.Nevra]namedPackage) (*utils.Nevra, bool) {
	// skip "phantom" package
	if strings.HasPrefix(nevraStr, "gpg-pubkey") {
		return nil, false
	}

	// Parse each NEVRA in the input
	nevra, err := utils.ParseNevra(nevraStr)
	if err != nil {
		utils.Log("nevra", nevraStr).Warn("Invalid nevra")
		return nil, false
	}

	// Check whether we have that NEVRA in DB
	currentNamedPackage := (*packagesByNEVRA)[*nevra]
	if currentNamedPackage.PackageID == 0 {
		utils.Log("nevra", nevraStr).Trace("Unknown package")
		return nil, false
	}
	return nevra, true
}

func updateDataChanged(currentNamedPackage *namedPackage, updateDataJSON []byte) bool {
	if bytes.Equal(updateDataJSON, currentNamedPackage.UpdateData.RawMessage) {
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

func createSystemPackage(nevra *utils.Nevra,
	updateData vmaas.UpdatesV2ResponseUpdateList,
	system *models.SystemPlatform,
	packagesByNEVRA *map[utils.Nevra]namedPackage) (systemPackagePtr *models.SystemPackage, updatesChanged bool) {
	updateDataJSON, err := vmaasResponse2UpdateDataJSON(&updateData)
	if err != nil {
		utils.Log("nevra", nevra.String()).Error("VMaaS updates response parsing failed")
		return nil, false
	}

	// Skip overwriting entries which have the same data as before
	currentNamedPackage := (*packagesByNEVRA)[*nevra]
	if !updateDataChanged(&currentNamedPackage, updateDataJSON) {
		return nil, false
	}

	systemPackage := models.SystemPackage{
		RhAccountID: system.RhAccountID,
		SystemID:    system.ID,
		PackageID:   currentNamedPackage.PackageID,
		UpdateData:  postgres.Jsonb{RawMessage: updateDataJSON},
		NameID:      currentNamedPackage.NameID,
	}
	return &systemPackage, true
}

func updateSystemPackages(tx *gorm.DB, system *models.SystemPlatform,
	packagesByNEVRA *map[utils.Nevra]namedPackage,
	updates map[string]vmaas.UpdatesV2ResponseUpdateList) (installed, updatable int, err error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-store"))

	if err := deleteOldSystemPackages(tx, system, packagesByNEVRA); err != nil {
		return 0, 0, err
	}

	toStore := make([]models.SystemPackage, 0, len(updates))
	for nevraStr, updateData := range updates {
		nevra, isValid := isValidNevra(nevraStr, packagesByNEVRA)
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
	pkgUpdates := make([]models.PackageUpdate, 0, len(updateData.GetAvailableUpdates()))
	for _, upData := range updateData.GetAvailableUpdates() {
		upNevra, err := utils.ParseNevra(upData.GetPackage())
		// Skip invalid nevras in updates list
		if err != nil {
			utils.Log("nevra", upData.Package).Warn("Invalid nevra")
			continue
		}
		// Create correct entry for each update in the list
		pkgUpdates = append(pkgUpdates, models.PackageUpdate{
			EVRA:     upNevra.EVRAString(),
			Advisory: upData.GetErratum(),
		})
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
	packagesByNEVRA *map[utils.Nevra]namedPackage) error {
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
	UpdateData postgres.Jsonb
}
