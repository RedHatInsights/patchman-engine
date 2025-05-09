package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/types"
	"app/base/utils"
	"app/base/vmaas"
	"app/tasks"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/pkg/errors"
)

const SyncBatchSize = 1000 // Should be < 5000

// Map advisory types received from vmaas. Done to support EPEL content
var advisoryTypeRemap = map[string]string{
	"newpackage": "enhancement",
}

func syncAdvisories(syncStart time.Time, modifiedSince *string) error {
	if vmaasClient == nil {
		panic("VMaaS client is nil")
	}

	iPage := 0
	iPageMax := 1
	advSyncStart := time.Now()
	for iPage <= iPageMax {
		errataResponse, err := downloadAndProcessErratasPage(iPage, modifiedSince)
		if err != nil {
			return errors.Wrap(err, "Erratas page download and process failed")
		}

		iPageMax = errataResponse.Pages
		utils.LogInfo("page", iPage, "pages", errataResponse.Pages, "count", len(errataResponse.ErrataList),
			"sync_duration", utils.SinceStr(syncStart, time.Second),
			"advisories_sync_duration", utils.SinceStr(advSyncStart, time.Second),
			"Downloaded advisories")
		iPage++
	}

	utils.LogInfo("modified_since", modifiedSince, "Advisories synced successfully")
	return nil
}

func getAdvisoryTypes() (map[string]int, error) {
	var advisoryTypesArr []models.AdvisoryType

	err := tasks.CancelableDB().Find(&advisoryTypesArr).Error
	if err != nil {
		return nil, errors.WithMessage(err, "Loading advisory types")
	}

	advisoryTypes := make(map[string]int, len(advisoryTypesArr))
	for _, t := range advisoryTypesArr {
		advisoryTypes[strings.ToLower(t.Name)] = t.ID
	}

	for from, to := range advisoryTypeRemap {
		advisoryTypes[from] = advisoryTypes[to]
	}

	return advisoryTypes, nil
}

func getAdvisorySeverities() (map[string]int, error) {
	var severitiesArr []models.AdvisorySeverity

	err := tasks.CancelableDB().Find(&severitiesArr).Error
	if err != nil {
		return nil, errors.WithMessage(err, "Loading advisory types")
	}

	severities := make(map[string]int, len(severitiesArr))
	for _, t := range severitiesArr {
		severities[strings.ToLower(t.Name)] = t.ID
	}
	return severities, nil
}

func getSeverityID(vmaasData *vmaas.ErrataResponseErrataList, severities map[string]int) *int {
	var severityID *int
	severity := vmaasData.Severity
	if severity != "" {
		if id, has := severities[strings.ToLower(severity)]; has {
			severityID = &id
		}
	}
	return severityID
}

func vmaasData2AdvisoryMetadata(errataName string, vmaasData vmaas.ErrataResponseErrataList,
	severities, advisoryTypes map[string]int) (*models.AdvisoryMetadata, error) {
	issued, err := time.Parse(types.Rfc3339NoTz, vmaasData.Issued)
	if err != nil {
		// try to parse timestamp with Z
		issued, err = time.Parse(time.RFC3339, vmaasData.Issued)
		if err != nil {
			return nil, errors.Wrap(err, "Invalid errata issued date")
		}
	}
	modified, success := checkUpdatedSummaryDescription(errataName, vmaasData)
	if !success {
		return nil, nil
	}

	packageData, cvesData, releaseVersionsData, err := getJSONFields(&vmaasData)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get JSON fields data")
	}

	advisory := models.AdvisoryMetadata{
		Name:            errataName,
		AdvisoryTypeID:  advisoryTypes[strings.ToLower(vmaasData.Type)],
		Description:     vmaasData.Description,
		Synopsis:        vmaasData.Synopsis,
		Summary:         vmaasData.Summary,
		Solution:        utils.EmptyToNil(vmaasData.Solution),
		SeverityID:      getSeverityID(&vmaasData, severities),
		CveList:         cvesData,
		PublicDate:      &issued,
		ModifiedDate:    &modified,
		URL:             utils.EmptyToNil(vmaasData.URL),
		PackageData:     packageData,
		RebootRequired:  vmaasData.RequiresReboot,
		ReleaseVersions: releaseVersionsData,
		Synced:          true,
	}
	return &advisory, nil
}

func checkUpdatedSummaryDescription(errataName string, vmaasData vmaas.ErrataResponseErrataList) (
	modified time.Time, success bool) {
	modified, err := time.Parse(types.Rfc3339NoTz, vmaasData.Updated)
	if err != nil {
		modified, err = time.Parse(time.RFC3339, vmaasData.Updated)
		if err != nil {
			utils.LogError("err", err.Error(), "erratum", errataName, "Invalid errata modified date")
			return time.Time{}, false
		}
	}

	if vmaasData.Description == "" || vmaasData.Summary == "" {
		utils.LogError("name", errataName, "An advisory without description or summary")
		return time.Time{}, false
	}
	return modified, true
}

func getJSONFields(vmaasData *vmaas.ErrataResponseErrataList) ([]byte, []byte, []byte, error) {
	packageData, err := getPackageData(vmaasData)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to get package data")
	}

	cvesData, err := sonic.Marshal(vmaasData.CveList)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Could not serialize CVEs data")
	}

	var releaseVersionsData []byte
	if vmaasData.ReleaseVersions != nil {
		releaseVersionsData, err = sonic.Marshal(vmaasData.ReleaseVersions)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "Could not serialize release_versions data")
		}
	}

	return packageData, cvesData, releaseVersionsData, nil
}

func getPackageData(vmaasData *vmaas.ErrataResponseErrataList) ([]byte, error) {
	packageData, err := sonic.Marshal(vmaasData.PackageList)
	if err != nil {
		return nil, errors.Wrap(err, "Could not serialize package data")
	}
	return packageData, nil
}

func parseAdvisories(data map[string]vmaas.ErrataResponseErrataList) (models.AdvisoryMetadataSlice, error) {
	var advisories models.AdvisoryMetadataSlice

	advisoryTypes, severities, err := getIDMaps()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to load IDs maps")
	}

	for errataName, vmaasData := range data {
		advisory, err := vmaasData2AdvisoryMetadata(errataName, vmaasData, severities, advisoryTypes)
		if err != nil {
			return nil, errors.Wrap(err, "advisory metadata item creating failed")
		}

		if advisory != nil {
			advisories = append(advisories, *advisory)
		}
	}
	return advisories, nil
}

func getIDMaps() (advisoryTypes, severities map[string]int, err error) {
	advisoryTypes, err = getAdvisoryTypes()
	if err != nil {
		return nil, nil, errors.Wrap(err, "advisory types map loading failed")
	}

	severities, err = getAdvisorySeverities()
	if err != nil {
		return nil, nil, errors.Wrap(err, "severities map loading failed")
	}
	return advisoryTypes, severities, nil
}

func storeAdvisories(data map[string]vmaas.ErrataResponseErrataList) error {
	advisories, err := parseAdvisories(data)
	if err != nil {
		return errors.WithMessage(err, "Parsing advisories")
	}

	if len(advisories) == 0 {
		return nil
	}

	var existingAdvisories models.AdvisoryMetadataSlice
	names := make([]string, 0, len(advisories))
	for _, a := range advisories {
		names = append(names, a.Name)
	}

	updateCols := []string{"description", "synopsis", "summary", "solution",
		"public_date", "modified_date", "url", "advisory_type_id", "severity_id", "cve_list", "package_data",
		"reboot_required", "release_versions", "synced"}

	tx := tasks.CancelableDB().Table("advisory_metadata")
	errSelect := tx.Where("name IN ?", names).Find(&existingAdvisories).Error
	if errSelect != nil {
		utils.LogWarn("err", errSelect, "couldn't find advisory_metadata for update")
	}

	inDBIDs := make(map[string]int64, len(existingAdvisories))
	for _, ea := range existingAdvisories {
		inDBIDs[ea.Name] = ea.ID
	}
	toUpdate := make(models.AdvisoryMetadataSlice, 0, len(existingAdvisories))
	toStore := make(models.AdvisoryMetadataSlice, 0, len(advisories)-len(existingAdvisories))
	for _, a := range advisories {
		if id, has := inDBIDs[a.Name]; has {
			a.ID = id
			toUpdate = append(toUpdate, a)
		} else {
			toStore = append(toStore, a)
		}
	}

	db := tasks.CancelableDB()
	for _, u := range toUpdate {
		if err := db.Table("advisory_metadata").Select(updateCols).Updates(u).Error; err != nil {
			utils.LogError("err", err, "couldn't update advisory_metadata")
		}
	}

	tx = database.OnConflictUpdate(db, "name", updateCols...)
	err = tx.CreateInBatches(&toStore, SyncBatchSize).Error
	if err != nil {
		return errors.WithMessage(err, "Storing advisories")
	}

	storeAdvisoriesCnt.WithLabelValues("success").Add(float64(len(data)))
	return nil
}

func downloadAndProcessErratasPage(iPage int, modifiedSince *string) (*vmaas.ErrataResponse, error) {
	errataResponse, err := vmaasErrataRequest(iPage, modifiedSince, tasks.AdvisoryPageSize)
	if err != nil {
		return nil, errors.Wrap(err, "Advisories sync failed on vmaas request")
	}

	if err = storeAdvisories(errataResponse.ErrataList); err != nil {
		storeAdvisoriesCnt.WithLabelValues("error").Add(float64(len(errataResponse.ErrataList)))
		return nil, errors.WithMessage(err, "Storing advisories")
	}
	return errataResponse, nil
}

func vmaasErrataRequest(iPage int, modifiedSince *string, pageSize int) (*vmaas.ErrataResponse, error) {
	errataRequest := vmaas.ErrataRequest{
		Page:          iPage,
		PageSize:      pageSize,
		ErrataList:    []string{".*"},
		ThirdParty:    utils.PtrBool(true),
		ModifiedSince: modifiedSince,
	}

	vmaasCallFunc := func() (interface{}, *http.Response, error) {
		vmaasData := vmaas.ErrataResponse{}
		resp, err := vmaasClient.Request(&base.Context, http.MethodPost, vmaasErratasURL, &errataRequest, &vmaasData)
		return &vmaasData, resp, err
	}

	vmaasDataPtr, err := utils.HTTPCallRetry(vmaasCallFunc, tasks.VmaasCallExpRetry, tasks.VmaasCallMaxRetries)
	if err != nil {
		vmaasCallCnt.WithLabelValues("error-download-errata").Inc()
		return nil, errors.Wrap(err, "Downloading erratas")
	}
	vmaasCallCnt.WithLabelValues("success").Inc()
	return vmaasDataPtr.(*vmaas.ErrataResponse), nil
}
