package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"crypto/sha256"
	"net/http"
	"time"

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
		iPageMax = pkgListResponse.Pages
		utils.Log("page", iPage, "pages", iPageMax, "count", len(pkgListResponse.PackageList),
			"sync_duration", utils.SinceStr(syncStart, time.Second),
			"packages_sync_duration", utils.SinceStr(pkgSyncStart, time.Second)).
			Info("Downloaded packages")
		iPage++
	}

	utils.Log("modified_since", modifiedSince).Info("Packages synced successfully")
	return nil
}

type nameIDandEvra struct {
	ID   int64
	Evra string
}

func stringPtr2Hash(str *string) *[]byte {
	if str == nil || *str == "" {
		return nil
	}
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

	err = storePkgListData(pkgListResponse.PackageList)
	if err != nil {
		return nil, errors.Wrap(err, "Packages data storing failed")
	}
	return pkgListResponse, nil
}

func vmaasPkgListRequest(iPage int, modifiedSince *string) (*vmaas.PkgListResponse, error) {
	request := vmaas.PkgListRequest{
		Page:          iPage,
		PageSize:      packagesPageSize,
		ModifiedSince: modifiedSince,
	}

	vmaasCallFunc := func() (interface{}, *http.Response, error) {
		vmaasData := vmaas.PkgListResponse{}
		resp, err := vmaasClient.Request(&base.Context, http.MethodPost, vmaasPkgListURL, &request, &vmaasData)
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

	if err = updatePackageNameSummary(database.Db, packageNameIDMap); err != nil {
		return errors.Wrap(err, "Updating package name summaries failed")
	}
	return nil
}

func storeStringsFromPkgListItems(tx *gorm.DB, vmaasData []vmaas.PkgListItem) error {
	stringMap := map[[32]byte]string{}
	for _, pkgListItem := range vmaasData {
		stringMap[sha256.Sum256([]byte(pkgListItem.Description))] = pkgListItem.Description
		stringMap[sha256.Sum256([]byte(pkgListItem.Summary))] = pkgListItem.Summary
	}

	strings := make([]models.String, 0, len(stringMap))
	for key, v := range stringMap {
		// need to allocate here, otherwise the slice references will point to stack space occupied by last element from
		// iteration.
		keySlice := make([]byte, 32)
		copy(keySlice, key[:])
		if v != "" {
			// don't store empty strings
			strings = append(strings, models.String{ID: keySlice, Value: v})
		}
	}

	utils.Log("strings", len(strings)).Info("Created package strings to store")
	tx = tx.Clauses(clause.OnConflict{
		DoNothing: true,
	})
	return tx.CreateInBatches(strings, chunkSize).Error
}

func storePackageNamesFromPkgListItems(tx *gorm.DB, vmaasData []vmaas.PkgListItem) (map[string]int, error) {
	packageNames, packageNameModels := getPackageArraysFromPkgListItems(tx, vmaasData)
	utils.Log("names", len(packageNames)).Info("Got package names")
	if len(packageNameModels) > 0 {
		tx = tx.Clauses(clause.OnConflict{
			DoNothing: true,
		}) // Insert missing
		err := tx.CreateInBatches(packageNameModels, chunkSize).Error
		if err != nil {
			return nil, errors.Wrap(err, "Bulk insert of package names failed")
		}
		utils.Log().Info("Package names stored")
	}

	packageNameIDMap, err := getPackageNameMap(tx, packageNames)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get package name map")
	}
	utils.Log("names", len(packageNameIDMap)).Info("Package names map loaded")
	return packageNameIDMap, nil
}

func getPackageArraysFromPkgListItems(tx *gorm.DB, pkgListItems []vmaas.PkgListItem) ([]string, []models.PackageName) {
	// get unique package names and their summaries
	namesMap := map[string]string{}
	for _, pkgListItem := range pkgListItems {
		nevra, err := utils.ParseNevra(pkgListItem.Nevra)
		if err != nil {
			utils.Log("nevra", pkgListItem.Nevra).Warn("Unable to parse package name")
			continue
		}
		namesMap[nevra.Name] = pkgListItem.Summary
	}

	var existingPkgsNames []models.PackageName
	nameArr := make([]string, 0, len(namesMap))
	for n := range namesMap {
		nameArr = append(nameArr, n)
	}
	// delete pkgs which exist in DB from namesMap
	if err := tx.Table("package_name").
		Where("name IN ?", nameArr).
		Find(&existingPkgsNames).
		Error; err != nil {
		utils.Log("err", err).Error("error in finding existing package names")
	}
	for _, ep := range existingPkgsNames {
		delete(namesMap, ep.Name)
	}
	pkgNames := make([]models.PackageName, 0, len(namesMap))
	for n := range namesMap {
		summary := namesMap[n]
		pkgNames = append(pkgNames, models.PackageName{Name: n, Summary: utils.EmptyToNil(&summary)})
	}
	return nameArr, pkgNames
}

func storePackageDetailsFrmPkgListItems(tx *gorm.DB, nameIDs map[string]int, pkgListItems []vmaas.PkgListItem) error {
	var toStore models.PackageSlice
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

	err := storeOrUpdate(tx, toStore)
	return err
}

//nolint:funlen
func storeOrUpdate(tx *gorm.DB, pkgs models.PackageSlice) error {
	var toUpdate models.PackageSlice

	nameIDEVRAs := make([][]interface{}, 0, len(pkgs))
	toStore := make(models.PackageSlice, 0, len(pkgs))
	updateIDs := make(map[nameIDandEvra]int64)
	for _, pkg := range pkgs {
		nameIDEVRAs = append(nameIDEVRAs, []interface{}{pkg.NameID, pkg.EVRA})
	}

	if err := tx.Where("(name_id, evra) IN ?", nameIDEVRAs).Find(&toUpdate).Error; err != nil {
		utils.Log("err", err).Warn("couldn't find packages for update")
	}
	for _, u := range toUpdate {
		updateIDs[nameIDandEvra{u.NameID, u.EVRA}] = u.ID
	}

	// set toUpdate and toStore
	toUpdate = make(models.PackageSlice, 0, len(pkgs))
	for _, p := range pkgs {
		if id, has := updateIDs[nameIDandEvra{p.NameID, p.EVRA}]; has {
			p.ID = id
			toUpdate = append(toUpdate, p)
		} else {
			toStore = append(toStore, p)
		}
	}

	// update packages
	var updErr error
	for _, u := range toUpdate {
		updErr = tx.Table("package").Select("description_hash", "summary_hash", "advisory_id").Updates(u).Error
	}
	if updErr != nil {
		storePackagesCnt.WithLabelValues("error").Add(float64(len(toUpdate)))
		updErr = errors.Wrap(updErr, "Packages update failed")
	}

	// insert packages
	tx = database.OnConflictUpdateMulti(tx, []string{"name_id", "evra"},
		"description_hash", "summary_hash", "advisory_id")
	if err := tx.CreateInBatches(toStore, chunkSize).Error; err != nil {
		storePackagesCnt.WithLabelValues("error").Add(float64(len(toStore)))
		return errors.Wrap(err, "Packages bulk insert failed")
	}
	if updErr != nil {
		storePackagesCnt.WithLabelValues("success").Add(float64(len(toStore)))
	} else {
		storePackagesCnt.WithLabelValues("success").Add(float64(len(pkgs)))
	}
	utils.Log().Info("Packages stored")
	return updErr
}

func getPackageFromPkgListItem(pkgListItem vmaas.PkgListItem, nameIDs map[string]int) *models.Package {
	nevraPtr, err := utils.ParseNevra(pkgListItem.Nevra)
	if err != nil {
		utils.Log("nevra", pkgListItem.Nevra).Error("Unable to parse nevra")
		return nil
	}

	descriptionStr := pkgListItem.Description
	summaryStr := pkgListItem.Summary
	pkg := models.Package{
		NameID:          int64(nameIDs[nevraPtr.Name]),
		EVRA:            nevraPtr.EVRAString(),
		DescriptionHash: stringPtr2Hash(&descriptionStr),
		SummaryHash:     stringPtr2Hash(&summaryStr),
		AdvisoryID:      nil, // we don't need to store package-advisory relation so far
		Synced:          true,
	}
	return &pkg
}

func updatePackageNameSummary(tx *gorm.DB, nameIDs map[string]int) error {
	pkgNameIDs := make([]int, 0, len(nameIDs))
	for _, val := range nameIDs {
		pkgNameIDs = append(pkgNameIDs, val)
	}
	err := tx.Exec(`UPDATE package_name pn
			          SET summary = latest.summary
					  FROM (SELECT DISTINCT ON (p.name_id) p.name_id, str.value as summary
							  FROM package p
							  JOIN strings str ON p.summary_hash = str.id
							 WHERE p.name_id in (?)
							 ORDER BY p.name_id, p.id desc) as latest
					WHERE pn.id = latest.name_id
					  AND latest.summary IS NOT NULL
					  AND (latest.summary != pn.summary OR pn.summary IS NULL)`,
		pkgNameIDs).Error
	if err == nil {
		utils.Log().Info("Package name summary updated")
	}
	return err
}
