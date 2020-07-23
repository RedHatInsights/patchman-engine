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

const saveChunkSize = 10 * 1024

func syncPackages(tx *gorm.DB, pkgs []string) error {
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

	idByName, dataByNevra, err := storePackageNames(tx, data.PackageList)
	if err != nil {
		return errors.Wrap(err, "Pkg names")
	}

	err = storePackageStrings(tx, dataByNevra)
	if err != nil {
		return errors.Wrap(err, "Pkg strings")
	}
	return storePackageDetails(tx, idByName, dataByNevra)
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
	errs := database.BulkInsertChunk(tx, pkgNames, saveChunkSize)
	if len(errs) > 0 {
		return nil, nil, errs[0]
	}
	// Load all to get IDs
	err := tx.Where("name in (?)", nameArr).Find(&pkgNames).Error
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
		strings = append(strings, models.String{ID: key, Value: v})
	}

	tx = tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	return database.BulkInsert(tx, strings)
}

func storePackageDetails(tx *gorm.DB, nameIDs map[string]int,
	data map[utils.Nevra]vmaas.PackagesResponsePackageList) error {
	toStore := make([]models.Package, 0, len(data))

	for nevra, data := range data {
		toStore = append(toStore, models.Package{
			NameID:          nameIDs[nevra.Name],
			EVRA:            nevra.EVRAString(),
			DescriptionHash: sha256.Sum256([]byte(data.Description)),
			SummaryHash:     sha256.Sum256([]byte(data.Summary)),
		})
	}

	tx = tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	return database.BulkInsert(tx, toStore)
}
