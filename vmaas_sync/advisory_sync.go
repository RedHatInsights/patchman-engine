package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"encoding/json"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pkg/errors"
	"strings"
	"time"
)

const SyncBatchSize = 1000 // Should be < 5000

func syncAdvisories(syncStart time.Time, modifiedSince *time.Time) error {
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

		iPageMax = int(errataResponse.GetPages())
		iPage++
		utils.Log("page", iPage, "pages", int(errataResponse.GetPages()), "count", len(errataResponse.GetErrataList()),
			"sync_duration", utils.SinceStr(syncStart), "advisories_sync_duration", utils.SinceStr(advSyncStart)).
			Debug("Downloaded advisories")
	}
	utils.Log().Info("Advisories synced successfully")
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
	severity := vmaasData.GetSeverity()
	if severity != "" {
		if id, has := severities[strings.ToLower(severity)]; has {
			severityID = &id
		}
	}
	return severityID
}

func vmaasData2AdvisoryMetadata(errataName string, vmaasData vmaas.ErrataResponseErrataList,
	severities, advisoryTypes map[string]int) (*models.AdvisoryMetadata, error) {
	modified, success := checkUpdatedSummaryDescription(errataName, vmaasData)
	if !success {
		return nil, nil
	}

	packageData, cvesData, err := getJSONFields(&vmaasData)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get JSON fields data")
	}

	advisory := models.AdvisoryMetadata{
		Name:           errataName,
		AdvisoryTypeID: advisoryTypes[strings.ToLower(vmaasData.GetType())],
		Description:    vmaasData.GetDescription(),
		Synopsis:       vmaasData.GetSynopsis(),
		Summary:        vmaasData.GetSummary(),
		Solution:       vmaasData.GetSolution(),
		SeverityID:     getSeverityID(&vmaasData, severities),
		CveList:        &postgres.Jsonb{RawMessage: cvesData},
		PublicDate:     vmaasData.GetIssued(),
		ModifiedDate:   modified,
		URL:            vmaasData.Url,
		PackageData:    &postgres.Jsonb{RawMessage: packageData},
	}
	return &advisory, nil
}

func checkUpdatedSummaryDescription(errataName string, vmaasData vmaas.ErrataResponseErrataList) (
	modified time.Time, success bool) {
	modified, err := base.ParseTime(vmaasData.GetUpdated())
	if err != nil {
		utils.Log("err", err.Error(), "erratum", errataName).Error("Invalid errata modified date")
		return time.Time{}, false
	}

	if vmaasData.GetDescription() == "" || vmaasData.GetSummary() == "" {
		utils.Log("name", errataName).Error("An advisory without description or summary")
		return time.Time{}, false
	}
	return modified, true
}

func getJSONFields(vmaasData *vmaas.ErrataResponseErrataList) ([]byte, []byte, error) {
	packageData, err := getPackageData(vmaasData)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to get package data")
	}

	cvesData, err := json.Marshal(vmaasData.CveList)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not serialize CVEs data")
	}
	return packageData, cvesData, nil
}

func getPackageData(vmaasData *vmaas.ErrataResponseErrataList) ([]byte, error) {
	packages := make(models.AdvisoryPackageData)
	for _, p := range vmaasData.GetPackageList() {
		nevra, err := utils.ParseNevra(p)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not parse nevra %s", p)
		}
		packages[nevra.Name] = nevra.EVRAString()
	}

	packageData, err := json.Marshal(packages)
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
		"public_date", "modified_date", "url", "advisory_type_id", "severity_id", "cve_list", "package_data")

	err = database.BulkInsertChunk(tx, advisories, SyncBatchSize)
	if err != nil {
		return errors.WithMessage(err, "Storing advisories")
	}

	storeAdvisoriesCnt.WithLabelValues("success").Add(float64(len(data)))
	return nil
}

func downloadAndProcessErratasPage(iPage int, modifiedSince *time.Time) (*vmaas.ErrataResponse, error) {
	errataResponse, err := vmaasErrataRequest(iPage, modifiedSince)
	if err != nil {
		return nil, errors.Wrap(err, "Advisories sync failed on vmaas request")
	}

	if err = storeAdvisories(errataResponse.GetErrataList()); err != nil {
		storeAdvisoriesCnt.WithLabelValues("error").Add(float64(len(errataResponse.GetErrataList())))
		return nil, errors.WithMessage(err, "Storing advisories")
	}
	return errataResponse, nil
}

func vmaasErrataRequest(iPage int, modifiedSince *time.Time) (*vmaas.ErrataResponse, error) {
	errataRequest := vmaas.ErrataRequest{
		Page:          utils.PtrFloat32(float32(iPage)),
		PageSize:      utils.PtrFloat32(float32(advisoryPageSize)),
		ErrataList:    []string{".*"},
		ModifiedSince: modifiedSince,
	}

	resp, _, err := vmaasClient.DefaultApi.AppErrataHandlerPostPost(base.Context).ErrataRequest(errataRequest).Execute()
	if err != nil {
		vmaasCallCnt.WithLabelValues("error-download-errata").Inc()
		return nil, errors.Wrap(err, "Downloading erratas")
	}
	vmaasCallCnt.WithLabelValues("success").Inc()
	return &resp, nil
}
