package listener

import (
	"app/base/content_sources"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"gorm.io/gorm"
)

func syncTemplateAdvisories(ctx context.Context, accountID int, templateID int64, templateUUID string) error {
	current, err := callCSTemplateAdvisories(ctx, templateUUID)
	if err != nil {
		return errors.Wrap(err, "fetching current template advisories from content-sources")
	}

	stored, err := lookUpTemplateAdvisories(database.DB, accountID, templateID)
	if err != nil {
		return errors.Wrap(err, "looking up stored template advisories")
	}

	toAdd, toRemove := diffTemplateAdvisories(current.AdvisoryIDs, stored)
	if len(toAdd) == 0 && len(toRemove) == 0 {
		utils.LogInfo("template_uuid", templateUUID, "template advisories unchanged")
		return nil
	}

	nameToID, err := lookUpAdvisoryMetadataIDs(toAdd)
	if err != nil {
		return errors.Wrap(err, "looking up advisory metadata IDs")
	}

	toInsert := buildTemplateAdvisoryRows(accountID, templateID, toAdd, nameToID)

	tx := database.DB.Begin()
	defer tx.Rollback()

	err = deleteOldTemplateAdvisories(tx, accountID, templateID, toRemove)
	if err != nil {
		return errors.Wrap(err, "deleting old template advisories")
	}

	if len(toInsert) > 0 {
		err = database.BulkInsert(tx, toInsert)
		if err != nil {
			return errors.Wrap(err, "bulk inserting added template advisories")
		}
	}

	utils.LogInfo(
		"template_uuid", templateUUID,
		"added_advisories", len(toInsert),
		"removed_advisories", len(toRemove),
		"template advisories synced",
	)

	return tx.Commit().Error
}

// Returns advisories currently stored for the template
func lookUpTemplateAdvisories(tx *gorm.DB, accountID int,
	templateID int64) (map[string]models.TemplateAdvisory, error) {
	var data []models.TemplateAdvisory
	err := tx.Preload("Advisory").
		Find(&data, "template_id = ? AND rh_account_id = ?", templateID, accountID).Error
	if err != nil {
		return nil, err
	}

	advisories := make(map[string]models.TemplateAdvisory, len(data))
	for _, ta := range data {
		advisories[ta.Advisory.Name] = ta
	}

	return advisories, nil
}

// Calculates the diff between template advisories from content sources and stored template advisories
func diffTemplateAdvisories(current []string, stored map[string]models.TemplateAdvisory) ([]string, []int64) {
	var toAdd []string
	var toRemove []int64

	// advisories from content sources
	currentSet := make(map[string]struct{}, len(current))
	for _, name := range current {
		currentSet[name] = struct{}{}
	}
	// prepare to remove if in DB but not in content sources
	for name, ta := range stored {
		if _, found := currentSet[name]; !found {
			toRemove = append(toRemove, ta.AdvisoryID)
		}
	}
	// prepare to add if in content sources but not in DB
	for name := range currentSet {
		if _, found := stored[name]; !found {
			toAdd = append(toAdd, name)
		}
	}

	return toAdd, toRemove
}

// Returns the advisory IDs from advisory_metadata that map to the given advisory names
func lookUpAdvisoryMetadataIDs(names []string) (map[string]int64, error) {
	metadata := make(models.AdvisoryMetadataSlice, 0, len(names))

	err := database.DB.Model(&models.AdvisoryMetadata{}).
		Where("name IN (?)", names).
		Select("id, name").
		Scan(&metadata).Error
	if err != nil {
		return nil, err
	}

	nameToID := make(map[string]int64, len(metadata))
	for _, am := range metadata {
		nameToID[am.Name] = am.ID
	}

	return nameToID, err
}

// Builds the rows to insert in template_advisory
func buildTemplateAdvisoryRows(accountID int, templateID int64, names []string,
	nameToID map[string]int64) models.TemplateAdvisorySlice {
	rows := make(models.TemplateAdvisorySlice, 0, len(names))
	for _, name := range names {
		advisoryID, ok := nameToID[name]
		if !ok {
			// log warning and skip if advisory not yet found in advisory_metadata
			// content sources might know about it before vmaas
			utils.LogWarn("template", templateID, "advisory", name, "not in advisory_metadata, skipping")
			continue
		}
		rows = append(rows, models.TemplateAdvisory{
			RhAccountID: accountID,
			TemplateID:  templateID,
			AdvisoryID:  advisoryID,
		})
	}
	return rows
}

// Deletes given template advisory relationships from template_advisory
func deleteOldTemplateAdvisories(tx *gorm.DB, accountID int, templateID int64, advisoryIDs []int64) error {
	if len(advisoryIDs) == 0 {
		return nil
	}

	err := tx.Where("rh_account_id = ? ", accountID).
		Where("template_id = ?", templateID).
		Where("advisory_id in (?)", advisoryIDs).
		Delete(&models.TemplateAdvisory{}).Error
	return err
}

func httpCallCSTemplateAdvisories(ctx context.Context, templateUUID string) (
	interface{}, *http.Response, error) {
	id := identity.GetIdentity(ctx)
	header, err := utils.EncodeXRHID(id.Identity)
	if err != nil {
		return nil, nil, err
	}

	if contentSourcesClient == nil {
		return nil, nil, errors.New("content sources client is nil")
	}
	client := *contentSourcesClient
	client.DefaultHeaders = map[string]string{"x-rh-identity": header}

	url := contentSourcesBaseURL + "/templates/" + templateUUID + "/advisories/ids"
	var resp content_sources.TemplateAdvisoryIDsResponse
	httpResp, err := client.Request(&ctx, http.MethodGet, url, nil, &resp)

	return &resp, httpResp, err
}

// Fetches advisories for a template.
// Returns an unpaginated list of advisory IDs (e.g. RHSA-1234:0001)
func callCSTemplateAdvisories(ctx context.Context,
	templateUUID string) (*content_sources.TemplateAdvisoryIDsResponse, error) {
	contentSourcesRespPtr, err := utils.HTTPCallRetry(
		func() (interface{}, *http.Response, error) {
			return httpCallCSTemplateAdvisories(ctx, templateUUID)
		},
		true,
		5,
		http.StatusServiceUnavailable)
	if err != nil {
		return nil, errors.Wrap(err, "content sources template advisories call failed")
	}
	return contentSourcesRespPtr.(*content_sources.TemplateAdvisoryIDsResponse), nil
}
