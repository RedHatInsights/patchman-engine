package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"crypto/sha256"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

const chunkSize = 10 * 1024

func syncPackages(tx *gorm.DB, advisoryIDs map[utils.Nevra]int, pkgs []string) error {
	vmaasData, err := vmaasRequestPackages(pkgs)
	if err != nil {
		return errors.Wrap(err, "Packages sync failed on vmaas request.")
	}

	err = storePackageData(tx, advisoryIDs, vmaasData.GetPackageList())
	if err != nil {
		return errors.Wrap(err, "Packages sync failed on vmaas request")
	}
	return nil
}

func vmaasRequestPackages(pkgs []string) (*vmaas.PackagesResponse, error) {
	utils.Log("count", len(pkgs)).Debug("Downloading packages...")
	query := vmaas.PackagesRequest{
		PackageList: pkgs,
	}
	vmaasData, _, err := vmaasClient.DefaultApi.AppPackagesHandlerPostPost(base.Context).
		PackagesRequest(query).Execute()
	if err != nil {
		return nil, errors.Wrap(err, "Vmaas packages request failed")
	}
	return &vmaasData, nil
}

func storePackageData(tx *gorm.DB, advisoryIDs map[utils.Nevra]int,
	vmaasData map[string]vmaas.PackagesResponsePackageList) error {
	utils.Log("count", len(vmaasData)).Debug("Storing packages...")
	idByName, dataByNevra, err := storePackageNames(tx, vmaasData)
	if err != nil {
		return errors.Wrap(err, "Packages names storing failed")
	}

	if err := storePackageStrings(tx, dataByNevra); err != nil {
		return errors.Wrap(err, "Package strings storing failed")
	}

	if err := storePackageDetails(tx, advisoryIDs, idByName, dataByNevra); err != nil {
		return errors.Wrap(err, "Package details store failed")
	}

	if err := tx.Exec("SELECT refresh_latest_packages_view()").Error; err != nil {
		return errors.Wrap(err, "Refreshing latest packages cache")
	}
	return nil
}

func storePackageNames(tx *gorm.DB, pkgs map[string]vmaas.PackagesResponsePackageList) (map[string]int,
	map[utils.Nevra]vmaas.PackagesResponsePackageList, error) {
	nameMap, byNevra := getPackageMaps(pkgs)
	nameArr, pkgNames := getPackageArrays(nameMap)
	tx = tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING") // Insert missing
	err := database.BulkInsertChunk(tx, pkgNames, chunkSize)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Bulk insert of package names failed")
	}

	idByName, err := getPackageNameMap(tx, nameArr)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Unable to get package name map")
	}
	return idByName, byNevra, nil
}

func getPackageNameMap(tx *gorm.DB, nameArr []string) (map[string]int, error) {
	// Load all to get IDs
	var pkgNamesLoaded []models.PackageName
	err := tx.Where("name in (?)", nameArr).Find(&pkgNamesLoaded).Error
	if err != nil {
		return nil, errors.Wrap(err, "Unable to load package names data")
	}
	idByName := map[string]int{}
	for _, p := range pkgNamesLoaded {
		idByName[p.Name] = p.ID
	}
	return idByName, nil
}

func getPackageArrays(nameMap map[string]bool) ([]string, []models.PackageName) {
	nameArr := make([]string, 0, len(nameMap))
	pkgNames := make([]models.PackageName, 0, len(nameMap))
	for n := range nameMap {
		nameArr = append(nameArr, n)
		pkgNames = append(pkgNames, models.PackageName{Name: n})
	}
	return nameArr, pkgNames
}

func getPackageMaps(pkgs map[string]vmaas.PackagesResponsePackageList) (
	map[string]bool, map[utils.Nevra]vmaas.PackagesResponsePackageList) {
	// We use map to deduplicate package names for DB insertion
	nameMap := map[string]bool{}
	byNevra := map[utils.Nevra]vmaas.PackagesResponsePackageList{}
	for pkg, detail := range pkgs {
		nevra, err := utils.ParseNevra(pkg)
		if err != nil {
			utils.Log("err", err, "nevra", pkg).Warn("Invalid nevra")
			continue
		}
		nameMap[nevra.Name] = true
		byNevra[*nevra] = detail
	}
	return nameMap, byNevra
}

func storePackageStrings(tx *gorm.DB, data map[utils.Nevra]vmaas.PackagesResponsePackageList) error {
	stringMap := map[[32]byte]string{}
	for _, r := range data {
		stringMap[sha256.Sum256([]byte(r.GetDescription()))] = r.GetDescription()
		stringMap[sha256.Sum256([]byte(r.GetSummary()))] = r.GetSummary()
	}

	strings := make([]models.String, 0, len(stringMap))
	for key, v := range stringMap {
		// need to allocate here, otherwise the slice references will point to stack space occupied by last element from
		// iteration.
		keySlice := make([]byte, 32)
		copy(keySlice, key[:])
		strings = append(strings, models.String{ID: keySlice, Value: v})
	}

	tx = tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	return database.BulkInsertChunk(tx, strings, chunkSize)
}

type packageID struct {
	NameID int
	EVRA   string
}

func storePackageDetails(tx *gorm.DB, advisoryIDs map[utils.Nevra]int, nameIDs map[string]int,
	vmaasNevraMap map[utils.Nevra]vmaas.PackagesResponsePackageList) error {
	names, evras := getNamesEvrasArrays(vmaasNevraMap)
	oldPackagesMap, err := getOldPackagesMap(evras, names)
	if err != nil {
		return errors.Wrap(err, "Unable to get old packages map")
	}

	inserts, updates := getInsertUpdatePackages(vmaasNevraMap, advisoryIDs, nameIDs, oldPackagesMap)
	if err = database.Db.Update(&updates).Error; err != nil {
		return errors.Wrap(err, "")
	}

	tx = tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	if err = database.BulkInsertChunk(tx, inserts, chunkSize); err != nil {
		return errors.Wrap(err, "Packages bulk insert failed")
	}
	return nil
}

func getInsertUpdatePackages(vmaasNevraMap map[utils.Nevra]vmaas.PackagesResponsePackageList,
	advisoryIDs map[utils.Nevra]int, nameIDs map[string]int, oldPackagesMap map[packageID]models.Package) (
	[]models.Package, []models.Package) {
	inserts := make([]models.Package, 0, len(vmaasNevraMap))
	updates := make([]models.Package, 0, len(vmaasNevraMap))
	for nevra, packageList := range vmaasNevraMap {
		nextPackage := getPackage(advisoryIDs, nevra, packageList, nameIDs)
		if nextPackage == nil {
			continue
		}

		if old, has := oldPackagesMap[packageID{nextPackage.NameID, nextPackage.EVRA}]; has {
			nextPackage.ID = old.ID
			updates = append(updates, *nextPackage)
		} else {
			inserts = append(inserts, *nextPackage)
		}
	}
	return inserts, updates
}

func getPackage(advisoryIDs map[utils.Nevra]int, nevra utils.Nevra, packageList vmaas.PackagesResponsePackageList,
	nameIDs map[string]int) *models.Package {
	if _, has := advisoryIDs[nevra]; !has {
		utils.Log("nevra", nevra.String()).Warn("Did not find matching advisories for nevra")
		return nil
	}

	description, summary := getDescriptionAndSummaryHashes(packageList)
	advisoryID := advisoryIDs[nevra]
	pkg := models.Package{
		NameID:          nameIDs[nevra.Name],
		EVRA:            nevra.EVRAString(),
		DescriptionHash: description,
		SummaryHash:     summary,
		AdvisoryID:      &advisoryID,
	}
	return &pkg
}

func getDescriptionAndSummaryHashes(packageList vmaas.PackagesResponsePackageList) (description, summary *[]byte) {
	descriptionBytes32 := sha256.Sum256([]byte(packageList.GetDescription()))
	summaryBytes32 := sha256.Sum256([]byte(packageList.GetSummary()))
	descriptionBytes := descriptionBytes32[:]
	summaryBytes := summaryBytes32[:]
	return &descriptionBytes, &summaryBytes
}

func getOldPackagesMap(evras []string, names []string) (map[packageID]models.Package, error) {
	var oldPackages []models.Package
	err := database.Db.
		Table("package p").
		Select("pn.name, p.*").
		Joins("JOIN package_name pn ON pn.id = p.name_id").
		Where("p.evra in (?)", evras).
		Where("pn.name in (?)", names).
		Find(&oldPackages).Error
	if err != nil {
		return nil, errors.Wrap(err, "Loading old packages")
	}

	oldPackagesMap := map[packageID]models.Package{}
	for _, o := range oldPackages {
		oldPackagesMap[packageID{o.NameID, o.EVRA}] = o
	}
	return oldPackagesMap, nil
}

func getNamesEvrasArrays(data map[utils.Nevra]vmaas.PackagesResponsePackageList) (names []string, evras []string) {
	names = make([]string, 0, len(data))
	evras = make([]string, 0, len(data))
	for nevra := range data {
		names = append(names, nevra.Name)
		evras = append(evras, nevra.EVRAString())
	}
	return names, evras
}
