package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"crypto/sha256"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/http"
	"time"
)

const chunkSize = 10 * 1024

func syncPackages(syncStart time.Time, modifiedSince *string) error {
	if vmaasClient == nil {
		panic("VMaaS client is nil")
	}

	iPage := 0
	iPageMax := 1
	pkgSyncStart := time.Now()
	for iPage <= iPageMax {
		pkgtreeResponse, err := downloadAndProcessPkgtreePage(iPage, modifiedSince)
		if err != nil {
			return errors.Wrap(err, "Pkgtree page download and process failed")
		}

		iPageMax = int(pkgtreeResponse.GetPages())
		iPage++
		utils.Log("page", iPage, "pages", int(pkgtreeResponse.GetPages()), "count", len(pkgtreeResponse.GetPackageNameList()),
			"sync_duration", utils.SinceStr(syncStart), "packages_sync_duration", utils.SinceStr(pkgSyncStart)).
			Debug("Downloaded packages")
	}
	utils.Log().Info("Packages synced successfully")
	return nil
}

func downloadAndProcessPkgtreePage(iPage int, modifiedSince *string) (*vmaas.PkgtreeResponse, error) {
	pkgtreeResponse, err := vmaasPkgtreeRequest(iPage, modifiedSince)
	if err != nil {
		return nil, errors.Wrap(err, "Packages sync failed on vmaas request")
	}

	err = storePackagesData(pkgtreeResponse.GetPackageNameList())
	if err != nil {
		return nil, errors.Wrap(err, "Packages data storing failed")
	}
	return pkgtreeResponse, nil
}

func storePackagesData(vmaasData map[string][]vmaas.PkgTreeItem) error {
	if err := storePackageStrings(database.Db, vmaasData); err != nil {
		return errors.Wrap(err, "Storing package strings failed")
	}

	packageNameIDMap, err := storePackageNames(database.Db, vmaasData)
	if err != nil {
		return errors.Wrap(err, "Storing package names failed")
	}

	if err = storePackageDetails(database.Db, packageNameIDMap, vmaasData); err != nil {
		return errors.Wrap(err, "Storing package details failed")
	}
	return nil
}

func vmaasPkgtreeRequest(iPage int, modifiedSince *string) (*vmaas.PkgtreeResponse, error) {
	errataRequest := vmaas.PkgtreeRequest{
		Page:               utils.PtrFloat32(float32(iPage)),
		PageSize:           utils.PtrFloat32(float32(packagesPageSize)),
		PackageNameList:    []string{".*"},
		ReturnSummary:      vmaas.PtrBool(true),
		ReturnDescription:  vmaas.PtrBool(true),
		ReturnRepositories: vmaas.PtrBool(false),
		ReturnErrata:       vmaas.PtrBool(false),
		ThirdParty:         vmaas.PtrBool(true),
		ModifiedSince:      modifiedSince,
	}

	vmaasCallFunc := func() (interface{}, *http.Response, error) {
		vmaasData, resp, err := vmaasClient.DefaultApi.AppPkgtreeHandlerV3PostPost(base.Context).
			PkgtreeRequest(errataRequest).Execute()
		return &vmaasData, resp, err
	}

	vmaasDataPtr, err := utils.HTTPCallRetry(base.Context, vmaasCallFunc, vmaasCallExpRetry, vmaasCallMaxRetries)
	if err != nil {
		vmaasCallCnt.WithLabelValues("error-download-errata").Inc()
		return nil, errors.Wrap(err, "Downloading erratas")
	}
	vmaasCallCnt.WithLabelValues("success").Inc()
	return vmaasDataPtr.(*vmaas.PkgtreeResponse), nil
}

func storePackageNames(tx *gorm.DB, vmaasData map[string][]vmaas.PkgTreeItem) (map[string]int, error) {
	packageNames, packageNameModels := getPackageArrays(vmaasData)
	utils.Log("names", len(packageNames)).Debug("Got package names")
	tx = tx.Clauses(clause.OnConflict{
		DoNothing: true,
	}) // Insert missing
	err := tx.CreateInBatches(packageNameModels, chunkSize).Error
	if err != nil {
		return nil, errors.Wrap(err, "Bulk insert of package names failed")
	}
	utils.Log().Debug("Package names stored")

	packageNameIDMap, err := getPackageNameMap(tx, packageNames)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get package name map")
	}
	utils.Log("names", len(packageNameIDMap)).Debug("Package names map loaded")
	return packageNameIDMap, nil
}

func getPackageArrays(nameMap map[string][]vmaas.PkgTreeItem) ([]string, []models.PackageName) {
	nameArr := make([]string, 0, len(nameMap))
	pkgNames := make([]models.PackageName, 0, len(nameMap))
	for n := range nameMap {
		nameArr = append(nameArr, n)
		pkgNames = append(pkgNames, models.PackageName{Name: n})
	}
	return nameArr, pkgNames
}

func storePackageStrings(tx *gorm.DB, vmaasData map[string][]vmaas.PkgTreeItem) error {
	stringMap := map[[32]byte]string{}
	for _, pkgTreeItems := range vmaasData {
		for _, pkgTreeItem := range pkgTreeItems {
			stringMap[sha256.Sum256([]byte(pkgTreeItem.GetDescription()))] = pkgTreeItem.GetDescription()
			stringMap[sha256.Sum256([]byte(pkgTreeItem.GetSummary()))] = pkgTreeItem.GetSummary()
		}
	}

	strings := make([]models.String, 0, len(stringMap))
	for key, v := range stringMap {
		// need to allocate here, otherwise the slice references will point to stack space occupied by last element from
		// iteration.
		keySlice := make([]byte, 32)
		copy(keySlice, key[:])
		strings = append(strings, models.String{ID: keySlice, Value: v})
	}

	utils.Log("strings", len(strings)).Debug("Created package strings to store")
	tx = tx.Clauses(clause.OnConflict{
		DoNothing: true,
	})
	return tx.CreateInBatches(strings, chunkSize).Error
}

type nameIDandEvra struct {
	ID   int
	Evra string
}

func storePackageDetails(tx *gorm.DB, nameIDs map[string]int, vmaasData map[string][]vmaas.PkgTreeItem) error {
	var toStore []models.Package
	var uniquePackages = make(map[nameIDandEvra]bool)
	for _, pkgTreeItems := range vmaasData {
		for _, pkgTreeItem := range pkgTreeItems {
			packageModel := getPackage(pkgTreeItem, nameIDs)
			if packageModel == nil {
				continue
			}

			key := nameIDandEvra{packageModel.NameID, packageModel.EVRA}
			if !uniquePackages[key] {
				toStore = append(toStore, *packageModel)
				uniquePackages[key] = true
			} else {
				utils.Log("nevra", pkgTreeItem.Nevra).Warn("Duplicit nevra found")
			}
		}
	}
	utils.Log("packages", len(toStore)).Debug("Collected packages to store")

	tx = database.OnConflictUpdateMulti(tx, []string{"name_id", "evra"},
		"description_hash", "summary_hash", "advisory_id")
	if err := tx.CreateInBatches(toStore, chunkSize).Error; err != nil {
		storePackagesCnt.WithLabelValues("error").Add(float64(len(toStore)))
		return errors.Wrap(err, "Packages bulk insert failed")
	}
	storePackagesCnt.WithLabelValues("success").Add(float64(len(toStore)))
	utils.Log().Debug("Packages stored")
	return nil
}

func getPackage(pkgTreeItem vmaas.PkgTreeItem, nameIDs map[string]int) *models.Package {
	nevraPtr, err := utils.ParseNevra(pkgTreeItem.Nevra)
	if err != nil {
		utils.Log("nevra", pkgTreeItem.Nevra).Error("Unable to parse nevra")
		return nil
	}

	descriptionStr := pkgTreeItem.GetDescription()
	summaryStr := pkgTreeItem.GetSummary()
	pkg := models.Package{
		NameID:          nameIDs[nevraPtr.Name],
		EVRA:            nevraPtr.EVRAString(),
		DescriptionHash: stringPtr2Hash(&descriptionStr),
		SummaryHash:     stringPtr2Hash(&summaryStr),
		AdvisoryID:      nil, // we don't need to store package-advisory relation so far
	}
	return &pkg
}

func stringPtr2Hash(str *string) *[]byte {
	bytes32 := sha256.Sum256([]byte(*str))
	bytes := bytes32[:]
	return &bytes
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
