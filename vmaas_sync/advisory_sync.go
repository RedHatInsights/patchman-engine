package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"encoding/json"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pkg/errors"
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
		issued, err := time.Parse(base.Rfc3339NoTz, v.Issued)
		if err != nil {
			utils.Log("err", err.Error(), "erratum", n).Error("Invalid errata issued date")
			continue
		}
		modified, err := time.Parse(base.Rfc3339NoTz, v.Updated)
		if err != nil {
			utils.Log("err", err.Error(), "erratum", n).Error("Invalid errata modified date")
			continue
		}

		if v.Description == "" || v.Summary == "" {
			utils.Log().Error("An advisory without description or summary")
			continue
		}
		var severityID *int
		if v.Severity != nil {
			if id, has := severities[strings.ToLower(*v.Severity)]; has {
				severityID = &id
			}
		}
		packages := make(models.AdvisoryPackageData)

		for _, p := range v.PackageList {
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

		advisory := models.AdvisoryMetadata{
			Name:           n,
			AdvisoryTypeID: advisoryTypes[strings.ToLower(v.Type)],
			Description:    v.Description,
			Synopsis:       v.Synopsis,
			Summary:        v.Summary,
			Solution:       v.Solution,
			SeverityID:     severityID,
			PublicDate:     issued,
			ModifiedDate:   modified,
			URL:            &v.Url,
			PackageData:    &postgres.Jsonb{RawMessage: packageData},
		}

		advisories = append(advisories, advisory)
	}
	return advisories, nil
}

func storeAdvisories(data map[string]vmaas.ErrataResponseErrataList) error {
	advisories, err := parseAdvisories(data)
	if err != nil {
		return errors.WithMessage(err, "Parsing advisories")
	}

	if advisories == nil || len(advisories) == 0 {
		return nil
	}

	tx := database.OnConflictUpdate(database.Db, "name", "description", "synopsis", "summary",
		"solution", "public_date", "modified_date", "url", "advisory_type_id", "severity_id", "package_data")
	errs := database.BulkInsertChunk(tx, advisories, SyncBatchSize)

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func syncAdvisories() error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, syncDuration)

	if vmaasClient == nil {
		panic("VMaaS client is nil")
	}

	pageIdx := 0
	maxPageIdx := 1

	for pageIdx <= maxPageIdx {
		opts := vmaas.AppErrataHandlerPostPostOpts{
			ErrataRequest: optional.NewInterface(vmaas.ErrataRequest{
				Page:          float32(pageIdx),
				PageSize:      float32(defaultPageSize),
				ErrataList:    []string{".*"},
				ModifiedSince: "",
			}),
		}

		data, _, err := vmaasClient.ErrataApi.AppErrataHandlerPostPost(base.Context, &opts)
		if err != nil {
			vmaasCallCnt.WithLabelValues("error-download-errata").Inc()
			return errors.WithMessage(err, "Downloading erratas")
		}
		vmaasCallCnt.WithLabelValues("success").Inc()

		maxPageIdx = int(data.Pages)
		pageIdx++

		utils.Log("count", len(data.ErrataList)).Debug("Downloaded advisories")

		err = storeAdvisories(data.ErrataList)
		if err != nil {
			storeAdvisoriesCnt.WithLabelValues("error").Add(float64(len(data.ErrataList)))
			return errors.WithMessage(err, "Storing advisories")
		}
		storeAdvisoriesCnt.WithLabelValues("success").Add(float64(len(data.ErrataList)))
	}
	return nil
}
