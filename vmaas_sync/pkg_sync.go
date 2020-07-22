package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

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
	for n := range nameMap {
		nameArr = append(nameArr, n)
	}

	var pkgNames []models.PackageName
	err := tx.Set("gorm:query_option", "FOR UPDATE OF package_name").
		Where("name in (?)", nameArr).Find(&pkgNames).Error
	if err != nil {
		return nil, nil, err
	}
	idByName := map[string]int{}
	for _, p := range pkgNames {
		idByName[p.Name] = p.ID
	}

	var newNames []models.PackageName
	for n := range nameMap {
		if _, has := idByName[n]; !has {
			newNames = append(newNames, models.PackageName{Name: n})
		}
	}
	err = database.BulkInsert(tx, newNames)
	if err != nil {
		return nil, nil, err
	}
	for _, n := range newNames {
		idByName[n.Name] = n.ID
	}
	return idByName, byNevra, nil
}

func storePackageDetails(tx *gorm.DB, nameIDs map[string]int,
	data map[utils.Nevra]vmaas.PackagesResponsePackageList) error {
	toStore := make([]models.Package, 0, len(data))

	for nevra, data := range data {
		toStore = append(toStore, models.Package{
			NameID:      nameIDs[nevra.Name],
			EVRA:        nevra.EVRAString(),
			Description: data.Description,
			Summary:     data.Summary,
		})
	}

	tx = tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	return database.BulkInsert(tx, toStore)
}
