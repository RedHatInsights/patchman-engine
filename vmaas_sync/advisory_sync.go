package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

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
		utils.Log("page", iPage, "pages", errataResponse.Pages, "count", len(errataResponse.ErrataList),
			"sync_duration", utils.SinceStr(syncStart, time.Second),
			"advisories_sync_duration", utils.SinceStr(advSyncStart, time.Second)).
			Info("Downloaded advisories")
		iPage++
	}

	advisoryCheckEnabled := utils.GetBoolEnvOrDefault("ENABLE_ADVISORIES_COUNT_CHECK", true)
	if modifiedSince != nil && advisoryCheckEnabled {
		err := checkAdvisoriesCount()
		if err != nil {
			return errors.Wrap(err, "Advisories check failed")
		}
	}

	utils.Log("modified_since", modifiedSince).Info("Advisories synced successfully")
	return nil
}

func getAdvisoryTypes() (map[string]int, error) {
	var advisoryTypesArr []models.AdvisoryType
	advisoryTypes := map[string]int{}

	err := database.Db.Find(&advisoryTypesArr).Error
	if err != nil {
		return nil, errors.WithMessage(err, "Loading advisory types")
	}

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
	severities := map[string]int{}

	err := database.Db.Find(&severitiesArr).Error
	if err != nil {
		return nil, errors.WithMessage(err, "Loading advisory types")
	}

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
	issued, err := time.Parse(base.Rfc3339NoTz, vmaasData.Issued)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid errata issued date")
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
		PublicDate:      issued,
		ModifiedDate:    modified,
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
	modified, err := time.Parse(base.Rfc3339NoTz, vmaasData.Updated)
	if err != nil {
		utils.Log("err", err.Error(), "erratum", errataName).Error("Invalid errata modified date")
		return time.Time{}, false
	}

	if vmaasData.Description == "" || vmaasData.Summary == "" {
		utils.Log("name", errataName).Error("An advisory without description or summary")
		return time.Time{}, false
	}
	return modified, true
}

func getJSONFields(vmaasData *vmaas.ErrataResponseErrataList) ([]byte, []byte, []byte, error) {
	packageData, err := getPackageData(vmaasData)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to get package data")
	}

	cvesData, err := json.Marshal(vmaasData.CveList)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Could not serialize CVEs data")
	}

	var releaseVersionsData []byte
	if vmaasData.ReleaseVersions != nil {
		releaseVersionsData, err = json.Marshal(vmaasData.ReleaseVersions)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "Could not serialize release_versions data")
		}
	}

	return packageData, cvesData, releaseVersionsData, nil
}

func getPackageData(vmaasData *vmaas.ErrataResponseErrataList) ([]byte, error) {
	packageData, err := json.Marshal(vmaasData.PackageList)
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

	if advisories == nil || len(advisories) == 0 {
		return nil
	}

	tx := database.OnConflictUpdate(database.Db, "name", "description", "synopsis", "summary", "solution",
		"public_date", "modified_date", "url", "advisory_type_id", "severity_id", "cve_list", "package_data",
		"reboot_required", "release_versions", "synced")

	err = tx.CreateInBatches(&advisories, SyncBatchSize).Error
	if err != nil {
		return errors.WithMessage(err, "Storing advisories")
	}

	storeAdvisoriesCnt.WithLabelValues("success").Add(float64(len(data)))
	return nil
}

func downloadAndProcessErratasPage(iPage int, modifiedSince *string) (*vmaas.ErrataResponse, error) {
	errataResponse, err := vmaasErrataRequest(iPage, modifiedSince, advisoryPageSize)
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

	vmaasDataPtr, err := utils.HTTPCallRetry(base.Context, vmaasCallFunc, vmaasCallExpRetry, vmaasCallMaxRetries)
	if err != nil {
		vmaasCallCnt.WithLabelValues("error-download-errata").Inc()
		return nil, errors.Wrap(err, "Downloading erratas")
	}
	vmaasCallCnt.WithLabelValues("success").Inc()
	return vmaasDataPtr.(*vmaas.ErrataResponse), nil
}

func checkAdvisoriesCount() error {
	var databaseAdvisoriesCount int64
	err := database.Db.Table("advisory_metadata").Count(&databaseAdvisoriesCount).Error
	if err != nil {
		return errors.Wrap(err, "Advisories check failed on db query")
	}

	errataResponse, err := vmaasErrataRequest(0, nil, 1)
	if err != nil {
		return errors.Wrap(err, "Advisories check failed on vmaas request")
	}

	errataCount := int64(errataResponse.Pages + 1)
	if databaseAdvisoriesCount != errataCount {
		mismatch := errataCount - databaseAdvisoriesCount
		advisoriesCountMismatch.Add(math.Abs(float64(mismatch)))
		utils.Log("mismatch", mismatch).Warning("Incremental advisories sync mismatch found!")
		err = syncAdvisories(time.Now(), nil)
		if err != nil {
			return errors.Wrap(err, "Full advisories sync failed")
		}
	}
	return nil
}
