package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"crypto/sha256"
	"net/http"
	"time"

	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		// Sync packages using /pkglist vmaas endpoint
		pkgListResponse, err := downloadAndProcessPkgListPage(iPage, modifiedSince)
		if err != nil {
			return errors.Wrap(err, "PkgList page download and process failed")
		}
		iPageMax = int(pkgListResponse.GetPages())
		utils.Log("page", iPage, "pages", iPageMax, "count", len(pkgListResponse.GetPackageList()),
			"sync_duration", utils.SinceStr(syncStart, time.Second),
			"packages_sync_duration", utils.SinceStr(pkgSyncStart, time.Second)).
			Info("Downloaded packages")
		iPage++
	}

	if modifiedSince != nil {
		checkPackagesCount()
	}

	utils.Log("modified_since", modifiedSince).Info("Packages synced successfully")
	return nil
}

type nameIDandEvra struct {
	ID   int
	Evra string
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

// Use /pkglist endpoint
func downloadAndProcessPkgListPage(iPage int, modifiedSince *string) (*vmaas.PkgListResponse, error) {
	pkgListResponse, err := vmaasPkgListRequest(iPage, modifiedSince)
	if err != nil {
		return nil, errors.Wrap(err, "Packages sync failed on vmaas request")
	}

	err = storePkgListData(pkgListResponse.GetPackageList())
	if err != nil {
		return nil, errors.Wrap(err, "Packages data storing failed")
	}
	return pkgListResponse, nil
}

func vmaasPkgListRequest(iPage int, modifiedSince *string) (*vmaas.PkgListResponse, error) {
	request := vmaas.PkgListRequest{
		Page:          utils.PtrFloat32(float32(iPage)),
		PageSize:      utils.PtrFloat32(float32(packagesPageSize)),
		ModifiedSince: modifiedSince,
	}

	vmaasCallFunc := func() (interface{}, *http.Response, error) {
		vmaasData, resp, err := vmaasClient.DefaultApi.VmaasWebappAppPkgListHandlerPostPost(base.Context).
			PkgListRequest(request).Execute()
		return &vmaasData, resp, err
	}

	vmaasDataPtr, err := utils.HTTPCallRetry(base.Context, vmaasCallFunc, vmaasCallExpRetry, vmaasCallMaxRetries)
	if err != nil {
		vmaasCallCnt.WithLabelValues("error-download-pkglist-response").Inc()
		return nil, errors.Wrap(err, "Downloading pkglist response")
	}
	vmaasCallCnt.WithLabelValues("success").Inc()
	return vmaasDataPtr.(*vmaas.PkgListResponse), nil
}

func storePkgListData(vmaasData []vmaas.PkgListItem) error {
	if err := storeStringsFromPkgListItems(database.Db, vmaasData); err != nil {
		return errors.Wrap(err, "Storing package strings failed")
	}

	packageNameIDMap, err := storePackageNamesFromPkgListItems(database.Db, vmaasData)
	if err != nil {
		return errors.Wrap(err, "Storing package names failed")
	}

	if err = storePackageDetailsFrmPkgListItems(database.Db, packageNameIDMap, vmaasData); err != nil {
		return errors.Wrap(err, "Storing package details failed")
	}
	return nil
}

func storeStringsFromPkgListItems(tx *gorm.DB, vmaasData []vmaas.PkgListItem) error {
	stringMap := map[[32]byte]string{}
	for _, pkgListItem := range vmaasData {
		stringMap[sha256.Sum256([]byte(pkgListItem.GetDescription()))] = pkgListItem.GetDescription()
		stringMap[sha256.Sum256([]byte(pkgListItem.GetSummary()))] = pkgListItem.GetSummary()
	}

	strings := make([]models.String, 0, len(stringMap))
	for key, v := range stringMap {
		// need to allocate here, otherwise the slice references will point to stack space occupied by last element from
		// iteration.
		keySlice := make([]byte, 32)
		copy(keySlice, key[:])
		strings = append(strings, models.String{ID: keySlice, Value: v})
	}

	utils.Log("strings", len(strings)).Info("Created package strings to store")
	tx = tx.Clauses(clause.OnConflict{
		DoNothing: true,
	})
	return tx.CreateInBatches(strings, chunkSize).Error
}

func storePackageNamesFromPkgListItems(tx *gorm.DB, vmaasData []vmaas.PkgListItem) (map[string]int, error) {
	packageNames, packageNameModels := getPackageArraysFromPkgListItems(vmaasData)
	utils.Log("names", len(packageNames)).Info("Got package names")
	tx = tx.Clauses(clause.OnConflict{
		DoNothing: true,
	}) // Insert missing
	err := tx.CreateInBatches(packageNameModels, chunkSize).Error
	if err != nil {
		return nil, errors.Wrap(err, "Bulk insert of package names failed")
	}
	utils.Log().Info("Package names stored")

	packageNameIDMap, err := getPackageNameMap(tx, packageNames)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get package name map")
	}
	utils.Log("names", len(packageNameIDMap)).Info("Package names map loaded")
	return packageNameIDMap, nil
}

func getPackageArraysFromPkgListItems(pkgListItems []vmaas.PkgListItem) ([]string, []models.PackageName) {
	// get unique package names
	namesMap := map[string]*bool{}
	for _, pkgListItem := range pkgListItems {
		nevra, err := utils.ParseNevra(pkgListItem.GetNevra())
		if err != nil {
			utils.Log("nevra", pkgListItem.GetNevra()).Warn("Unable to parse package name")
			continue
		}
		namesMap[nevra.Name] = nil
	}

	nameArr := make([]string, 0, len(namesMap))
	pkgNames := make([]models.PackageName, 0, len(namesMap))
	for n := range namesMap {
		nameArr = append(nameArr, n)
		pkgNames = append(pkgNames, models.PackageName{Name: n})
	}
	return nameArr, pkgNames
}

func storePackageDetailsFrmPkgListItems(tx *gorm.DB, nameIDs map[string]int, pkgListItems []vmaas.PkgListItem) error {
	var toStore []models.Package
	var uniquePackages = make(map[nameIDandEvra]bool)
	for _, pkgListItem := range pkgListItems {
		packageModel := getPackageFromPkgListItem(pkgListItem, nameIDs)
		if packageModel == nil {
			continue
		}

		key := nameIDandEvra{packageModel.NameID, packageModel.EVRA}
		if !uniquePackages[key] {
			toStore = append(toStore, *packageModel)
			uniquePackages[key] = true
		} else {
			utils.Log("nevra", pkgListItem.Nevra).Warn("Duplicit nevra found")
		}
	}
	utils.Log("packages", len(toStore)).Info("Collected packages to store")

	tx = database.OnConflictUpdateMulti(tx, []string{"name_id", "evra"},
		"description_hash", "summary_hash", "advisory_id")
	if err := tx.CreateInBatches(toStore, chunkSize).Error; err != nil {
		storePackagesCnt.WithLabelValues("error").Add(float64(len(toStore)))
		return errors.Wrap(err, "Packages bulk insert failed")
	}
	storePackagesCnt.WithLabelValues("success").Add(float64(len(toStore)))
	utils.Log().Info("Packages stored")
	return nil
}

func getPackageFromPkgListItem(pkgListItem vmaas.PkgListItem, nameIDs map[string]int) *models.Package {
	nevraPtr, err := utils.ParseNevra(pkgListItem.GetNevra())
	if err != nil {
		utils.Log("nevra", pkgListItem.Nevra).Error("Unable to parse nevra")
		return nil
	}

	descriptionStr := pkgListItem.GetDescription()
	summaryStr := pkgListItem.GetSummary()
	pkg := models.Package{
		NameID:          nameIDs[nevraPtr.Name],
		EVRA:            nevraPtr.EVRAString(),
		DescriptionHash: stringPtr2Hash(&descriptionStr),
		SummaryHash:     stringPtr2Hash(&summaryStr),
		AdvisoryID:      nil, // we don't need to store package-advisory relation so far
	}
	return &pkg
}

func checkPackagesCount() {
	packagesCheckEnabled := utils.GetBoolEnvOrDefault("ENABLE_PACKAGES_COUNT_CHECK", true)
	if !packagesCheckEnabled {
		return
	}

	var dbPkgCount int64
	err := database.Db.Table("package").Count(&dbPkgCount).Error
	if err != nil {
		utils.Log("err", err.Error()).Error("Packages check failed on db query")
		return
	}

	response, err := vmaasPkgListRequest(0, nil)
	if err != nil {
		utils.Log("err", err.Error()).Error("Packages check failed on vmaas request")
	}

	vmaasPkgCount := int64(response.GetTotal())
	if vmaasPkgCount <= dbPkgCount {
		utils.Log("vmaas-count", vmaasPkgCount, "patch-db-count", dbPkgCount).Info("Packages sync check OK")
		return
	}
	utils.Log("vmaas-count", vmaasPkgCount, "patch-db-count", dbPkgCount).Info("Running full packages sync")
	err = syncPackages(time.Now(), nil)
	if err != nil {
		utils.Log("err", err.Error()).Error("Full advisories sync failed")
	}
}
