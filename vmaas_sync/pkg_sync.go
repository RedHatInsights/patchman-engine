package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"crypto/sha256"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

const chunkSize = 10 * 1024

func syncPackages(tx *gorm.DB, advisoryIDs map[utils.Nevra]int, pkgs []string) error {
	utils.Log("count", len(pkgs)).Debug("Downloading packages...")
	query := vmaas.PackagesRequest{
		PackageList: pkgs,
	}
	opts := vmaas.AppPackagesHandlerPostPostOpts{
		PackagesRequest: optional.NewInterface(query),
	}
	data, _, err := vmaasClient.PackagesApi.AppPackagesHandlerPostPost(base.Context, &opts)
	if err != nil {
		return errors.Wrap(err, "Get packages")
	}

	utils.Log("count", len(pkgs)).Debug("Storing packages...")
	idByName, dataByNevra, err := storePackageNames(tx, data.PackageList)
	if err != nil {
		return errors.Wrap(err, "Pkg names")
	}

	err = storePackageStrings(tx, dataByNevra)
	if err != nil {
		return errors.Wrap(err, "Pkg strings")
	}
	err = storePackageDetails(tx, advisoryIDs, idByName, dataByNevra)
	if err != nil {
		return errors.Wrap(err, "Pkg details")
	}
	return errors.Wrap(refreshPackageCache(tx), "Refreshing latest packages cache")
}

func storePackageNames(tx *gorm.DB, pkgs map[string]vmaas.PackagesResponsePackageList) (map[string]int,
	map[utils.Nevra]vmaas.PackagesResponsePackageList, error) {
	// We use map to deduplicate package names for DB insertion
	nameMap := make(map[string]bool)
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
	nameArr := make([]string, 0, len(nameMap))
	pkgNames := make([]models.PackageName, 0, len(nameMap))
	for n := range nameMap {
		nameArr = append(nameArr, n)
		pkgNames = append(pkgNames, models.PackageName{Name: n})
	}
	// Insert missing
	tx = tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	err := database.BulkInsertChunk(tx, pkgNames, chunkSize)
	if err != nil {
		return nil, nil, err
	}
	// Load all to get IDs
	err = tx.Where("name in (?)", nameArr).Find(&pkgNames).Error
	if err != nil {
		return nil, nil, err
	}
	idByName := map[string]int{}
	for _, p := range pkgNames {
		idByName[p.Name] = p.ID
	}
	return idByName, byNevra, nil
}

func storePackageStrings(tx *gorm.DB, data map[utils.Nevra]vmaas.PackagesResponsePackageList) error {
	stringMap := map[[32]byte]string{}
	for _, r := range data {
		stringMap[sha256.Sum256([]byte(r.Description))] = r.Description
		stringMap[sha256.Sum256([]byte(r.Summary))] = r.Summary
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

func storePackageDetails(tx *gorm.DB, advisoryIDs map[utils.Nevra]int, nameIDs map[string]int,
	data map[utils.Nevra]vmaas.PackagesResponsePackageList) error {
	toStore := make([]models.Package, 0, len(data))

	for nevra, data := range data {
		desc := sha256.Sum256([]byte(data.Description))
		sum := sha256.Sum256([]byte(data.Summary))

		if _, has := advisoryIDs[nevra]; !has {
			utils.Log("nevra", nevra.String()).Warn("Did not find matching advisories for nevra")
			continue
		}
		toStore = append(toStore, models.Package{
			NameID:          nameIDs[nevra.Name],
			EVRA:            nevra.EVRAString(),
			DescriptionHash: desc[:],
			SummaryHash:     sum[:],
			AdvisoryID:      advisoryIDs[nevra],
		})
	}

	tx = tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	return database.BulkInsertChunk(tx, toStore, chunkSize)
}

func refreshPackageCache(tx *gorm.DB) error {
	return tx.Exec("REFRESH MATERIALIZED VIEW package_latest_cache").Error
}
