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
	"modernc.org/mathutil"
	"strings"
	"time"
)

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

// nolint: funlen
func parseAdvisories(data map[string]vmaas.ErrataResponseErrataList) (models.AdvisoryMetadataSlice, error) {
	var advisories models.AdvisoryMetadataSlice

	advisoryTypes, err := getAdvisoryTypes()
	if err != nil {
		return nil, err
	}
	severities, err := getAdvisorySeverities()
	if err != nil {
		return nil, err
	}

	for n, v := range data {
		// TODO: Should we skip or report invalid erratas ?
		issued := v.GetIssued()
		modified, err := time.Parse(base.Rfc3339NoTz, v.GetUpdated())
		if err != nil {
			utils.Log("err", err.Error(), "erratum", n).Error("Invalid errata modified date")
			continue
		}

		_, descriptionOk := v.GetDescriptionOk()
		_, summaryOk := v.GetSummaryOk()
		if !descriptionOk || !summaryOk {
			utils.Log("name", n).Error("An advisory without description or summary")
			continue
		}
		var severityID *int
		severity, severityOk := v.GetSeverityOk()
		if severityOk {
			if id, has := severities[strings.ToLower(*severity)]; has {
				severityID = &id
			}
		}
		packages := make(models.AdvisoryPackageData)

		for _, p := range v.GetPackageList() {
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

		cvesData, err := json.Marshal(v.CveList)
		if err != nil {
			return nil, errors.Wrap(err, "Could not serialize CVEs data")
		}

		advisory := models.AdvisoryMetadata{
			Name:           n,
			AdvisoryTypeID: advisoryTypes[strings.ToLower(v.GetType())],
			Description:    v.GetDescription(),
			Synopsis:       v.GetSynopsis(),
			Summary:        v.GetSummary(),
			Solution:       v.GetSolution(),
			SeverityID:     severityID,
			CveList:        &postgres.Jsonb{RawMessage: cvesData},
			PublicDate:     issued,
			ModifiedDate:   modified,
			URL:            v.Url,
			PackageData:    &postgres.Jsonb{RawMessage: packageData},
		}

		advisories = append(advisories, advisory)
	}
	return advisories, nil
}

func storeAdvisories(data map[string]vmaas.ErrataResponseErrataList) (map[string]int, error) {
	advisories, err := parseAdvisories(data)
	if err != nil {
		return nil, errors.WithMessage(err, "Parsing advisories")
	}

	if advisories == nil || len(advisories) == 0 {
		return nil, nil
	}

	tx := database.OnConflictUpdate(database.Db, "name", "description", "synopsis", "summary", "solution",
		"public_date", "modified_date", "url", "advisory_type_id", "severity_id", "cve_list", "package_data")

	err = database.BulkInsertChunk(tx, advisories, SyncBatchSize)
	if err != nil {
		return nil, errors.WithMessage(err, "Storing advisories")
	}

	advisoryIDs := make(map[string]int)
	for _, a := range advisories {
		advisoryIDs[a.Name] = a.ID
	}
	return advisoryIDs, nil
}

// nolint: funlen
func syncAdvisories() error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, syncDuration)

	if vmaasClient == nil {
		panic("VMaaS client is nil")
	}

	pageIdx := 0
	maxPageIdx := 1
	modifiedSince := time.Time{}

	for pageIdx <= maxPageIdx {
		errataRequest := vmaas.ErrataRequest{
			Page:          vmaas.PtrFloat32(float32(pageIdx)),
			PageSize:      vmaas.PtrFloat32(float32(advisoryPageSize)),
			ErrataList:    []string{".*"},
			ModifiedSince: &modifiedSince,
		}

		data, _, err := vmaasClient.DefaultApi.AppErrataHandlerPostPost(base.Context).ErrataRequest(errataRequest).Execute()
		if err != nil {
			vmaasCallCnt.WithLabelValues("error-download-errata").Inc()
			return errors.WithMessage(err, "Downloading erratas")
		}
		vmaasCallCnt.WithLabelValues("success").Inc()

		maxPageIdx = int(data.GetPages())
		pageIdx++

		utils.Log("count", len(data.GetErrataList())).Debug("Downloaded advisories")

		advisoryIDs, err := storeAdvisories(data.GetErrataList())
		if err != nil {
			storeAdvisoriesCnt.WithLabelValues("error").Add(float64(len(data.GetErrataList())))
			return errors.WithMessage(err, "Storing advisories")
		}
		storeAdvisoriesCnt.WithLabelValues("success").Add(float64(len(data.GetErrataList())))

		packages := []string{}
		// Map from package to AdvisoryID
		packageAdvisories := map[utils.Nevra]int{}
		for name, erratum := range data.GetErrataList() {
			if len(erratum.GetPackageList()) == 0 {
				continue
			}
			for _, p := range erratum.GetPackageList() {
				nevra, err := utils.ParseNevra(p)
				if err != nil {
					continue
				}

				packages = append(packages, p)
				packageAdvisories[*nevra] = advisoryIDs[name]
			}
		}

		for len(packages) > 0 {
			currentPageSize := mathutil.Min(packagesPageSize, len(packages))
			page := packages[0:currentPageSize]
			err = syncPackages(database.Db, packageAdvisories, page)
			if err != nil {
				storePackagesCnt.WithLabelValues("error").Add(float64(len(page)))
				return errors.WithMessage(err, "Storing packages")
			}
			storePackagesCnt.WithLabelValues("success").Add(float64(len(page)))
			packages = packages[currentPageSize:]
		}
	}
	utils.Log().Info("Advisories synced successfully")
	return nil
}
