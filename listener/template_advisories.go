package listener

import (
	"app/base/api"
	"app/base/content_sources"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/pkg/errors"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func syncTemplateAdvisories(ctx context.Context, accountID int, templateID int64, templateUUID, orgID string) error {
	current, err := callCSTemplateAdvisories(ctx, templateUUID)
	if err != nil {
		if errors.Is(err, errContentSourcesTemplateNotFound) {
			utils.LogWarn("template_uuid", templateUUID, "template not found in content sources, skipping", err.Error())
			return nil
		}
		return errors.Wrap(err, "fetching current template advisories from content-sources")
	}
	return applyTemplateAdvisories(ctx, accountID, templateID, templateUUID, orgID, current.AdvisoryIDs)
}

// applyTemplateAdvisories diffs advisory names against template_advisory and writes the change.
// advisoryIDs is the full snapshot. Nil and empty both mean no advisories.
func applyTemplateAdvisories(ctx context.Context, accountID int, templateID int64, templateUUID, orgID string,
	advisoryIDs []string) error {
	stored, err := lookUpTemplateAdvisories(database.DB(ctx), accountID, templateID)
	if err != nil {
		return errors.Wrap(err, "looking up stored template advisories")
	}

	toAdd, toRemove := diffTemplateAdvisories(advisoryIDs, stored)
	if len(toAdd) == 0 && len(toRemove) == 0 {
		utils.LogInfo("template_uuid", templateUUID, "template advisories unchanged")
		return nil
	}

	nameToID, err := lookUpAdvisoryMetadataIDs(ctx, toAdd)
	if err != nil {
		return errors.Wrap(err, "looking up advisory metadata IDs")
	}

	toInsert := buildTemplateAdvisoryRows(accountID, templateID, toAdd, nameToID)

	tx := database.DB(ctx).Begin()
	defer tx.Rollback()

	err = deleteOldTemplateAdvisories(tx, accountID, templateID, toRemove)
	if err != nil {
		return errors.Wrap(err, "deleting old template advisories")
	}

	if len(toInsert) > 0 {
		txOnConflict := tx.Clauses(clause.OnConflict{
			DoNothing: true,
		})
		err = database.BulkInsert(txOnConflict, toInsert)
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

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	if err := SendTemplateRecalc(ctx, accountID, orgID, templateID); err != nil {
		utils.LogError("err", err.Error(), "template_id", templateID, "account_id", accountID,
			"Template recalc lookup failed")
	}
	return nil
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

// Same placeholder shape as evaluator.storeMissingAdvisories. VMaaS sync overwrites it once the advisory is known.
var rhelAdvisoryName = regexp.MustCompile(`^RH.A-`)

const thirdPartyAdvisoryText = "Not Available for 3rd party systems"

// Returns advisory_metadata IDs for the given names.
// Names not yet stored are inserted as unsynced placeholders. Content can publish advisories before VMaaS sync.
func lookUpAdvisoryMetadataIDs(ctx context.Context, names []string) (map[string]int64, error) {
	nameToID, err := selectAdvisoryMetadataIDs(ctx, names)
	if err != nil {
		return nil, err
	}

	missing := missingAdvisoryNames(names, nameToID)
	if len(missing) == 0 {
		return nameToID, nil
	}

	if err := insertMissingAdvisoryMetadata(ctx, missing); err != nil {
		return nil, errors.Wrap(err, "creating missing advisory_metadata")
	}

	created, err := selectAdvisoryMetadataIDs(ctx, missing)
	if err != nil {
		return nil, err
	}
	for name, id := range created {
		nameToID[name] = id
	}
	return nameToID, nil
}

func selectAdvisoryMetadataIDs(ctx context.Context, names []string) (map[string]int64, error) {
	nameToID := make(map[string]int64, len(names))
	if len(names) == 0 {
		return nameToID, nil
	}

	metadata := make(models.AdvisoryMetadataSlice, 0, len(names))
	err := database.DB(ctx).Model(&models.AdvisoryMetadata{}).
		Where("name IN (?)", names).
		Select("id, name").
		Scan(&metadata).Error
	if err != nil {
		return nil, err
	}
	for _, am := range metadata {
		nameToID[am.Name] = am.ID
	}
	return nameToID, nil
}

func missingAdvisoryNames(names []string, nameToID map[string]int64) []string {
	missing := make([]string, 0)
	for _, name := range names {
		if _, found := nameToID[name]; found {
			continue
		}
		missing = append(missing, name)
	}
	return missing
}

func advisoryPlaceholderText(name string) string {
	if rhelAdvisoryName.MatchString(name) {
		return fmt.Sprintf("https://access.redhat.com/errata/%s", name)
	}
	return thirdPartyAdvisoryText
}

func insertMissingAdvisoryMetadata(ctx context.Context, names []string) error {
	toStore := make(models.AdvisoryMetadataSlice, 0, len(names))
	for _, name := range names {
		text := advisoryPlaceholderText(name)
		toStore = append(toStore, models.AdvisoryMetadata{
			Name:           name,
			Description:    text,
			Synopsis:       text,
			Summary:        text,
			AdvisoryTypeID: 0,
			RebootRequired: false,
			Synced:         false,
		})
	}
	return database.DB(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(&toStore, 500).Error
}

// Builds the rows to insert in template_advisory
func buildTemplateAdvisoryRows(accountID int, templateID int64, names []string,
	nameToID map[string]int64) models.TemplateAdvisorySlice {
	rows := make(models.TemplateAdvisorySlice, 0, len(names))
	for _, name := range names {
		advisoryID, ok := nameToID[name]
		if !ok {
			// missing name even after lookUpAdvisoryMetadataIDs, likely db error
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
	client := &api.Client{
		HTTPClient:     contentSourcesClient.HTTPClient,
		Debug:          contentSourcesClient.Debug,
		DefaultHeaders: map[string]string{"x-rh-identity": header},
	}

	url := contentSourcesBaseURL + "/templates/" + templateUUID + "/advisories/ids"
	var resp content_sources.TemplateAdvisoryIDsResponse
	httpResp, err := client.Request(&ctx, http.MethodGet, url, nil, &resp)
	if err != nil && httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return &resp, httpResp, errors.Wrap(errContentSourcesTemplateNotFound, err.Error())
	}

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
