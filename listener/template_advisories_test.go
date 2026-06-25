package listener

import (
	"app/base/content_sources"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"context"
	"testing"

	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"github.com/stretchr/testify/assert"
)

func setupContentSourcesClient(t *testing.T) {
	originalAddress := utils.CoreCfg.ContentSourcesAddress
	address := utils.Getenv("CONTENT_SOURCES_ADDRESS", "http://platform:9001")
	utils.CoreCfg.ContentSourcesAddress = address
	contentSourcesClient = content_sources.CreateContentSourcesClient()
	contentSourcesBaseURL = address + "/api/content-sources/v1"
	t.Cleanup(func() {
		utils.CoreCfg.ContentSourcesAddress = originalAddress
		contentSourcesClient = nil
		contentSourcesBaseURL = ""
	})
}

func TestCallCSTemplateAdvisories(t *testing.T) {
	setupContentSourcesClient(t)

	templateUUID := "99900000-0000-0000-0000-000000000001"
	orgID := "org_1"
	ctx := identity.WithIdentity(context.Background(), utils.XRHIDForOrg(orgID))
	result, err := callCSTemplateAdvisories(ctx, templateUUID)

	assert.NoError(t, err)
	assert.Equal(t, []string{"RH-1", "RH-3"}, result.AdvisoryIDs)
}

func TestDiffTemplateAdvisories(t *testing.T) {
	stored := map[string]models.TemplateAdvisory{
		"RH-1": {AdvisoryID: 1, Advisory: models.AdvisoryMetadata{Name: "RH-1"}},
		"RH-2": {AdvisoryID: 2, Advisory: models.AdvisoryMetadata{Name: "RH-2"}},
	}
	fromContentSources := []string{"RH-1", "RH-3"}

	toAdd, toRemove := diffTemplateAdvisories(fromContentSources, stored)
	assert.Equal(t, []string{"RH-3"}, toAdd)
	assert.Equal(t, []int64{2}, toRemove)
}

func TestLookUpAdvisoryMetadataIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	advisoryNames := []string{"RH-1", "RH-2"}
	nameToID, err := lookUpAdvisoryMetadataIDs(advisoryNames)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(nameToID))
	assert.Equal(t, int64(1), nameToID["RH-1"])
	assert.Equal(t, int64(2), nameToID["RH-2"])
}

func TestBuildTemplateAdvisoryRows(t *testing.T) {
	nameToID := map[string]int64{
		"RH-1": 1,
		"RH-2": 2,
	}
	rows := buildTemplateAdvisoryRows(1, 42, []string{"RH-1", "RH-2"}, nameToID)
	assert.Equal(t, 2, len(rows))
	assert.Equal(t, models.TemplateAdvisory{
		RhAccountID: 1,
		TemplateID:  42,
		AdvisoryID:  1,
	}, rows[0])
	assert.Equal(t, models.TemplateAdvisory{
		RhAccountID: 1,
		TemplateID:  42,
		AdvisoryID:  2,
	}, rows[1])
}

func TestBuildTemplateAdvisoryRows_SkipMissing(t *testing.T) {
	nameToID := map[string]int64{"RH-1": 1}
	rows := buildTemplateAdvisoryRows(1, 42, []string{"RH-1", "missing"}, nameToID)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, models.TemplateAdvisory{
		RhAccountID: 1,
		TemplateID:  42,
		AdvisoryID:  1,
	}, rows[0])
}

func TestLookUpTemplateAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	templateID := int64(1)
	advisoryIDs := []int64{1, 2}

	database.CreateTemplateAdvisories(t, accountID, templateID, advisoryIDs)
	database.CheckTemplateAdvisories(t, templateID, advisoryIDs)
	defer database.DeleteTemplateAdvisories(t, templateID, advisoryIDs)

	templateAdvisories, err := lookUpTemplateAdvisories(database.DB, accountID, templateID)
	assert.Nil(t, err)
	assert.NotNil(t, templateAdvisories)
	assert.Equal(t, 2, len(templateAdvisories))
	assert.Equal(t, "RH-1", (templateAdvisories)["RH-1"].Advisory.Name)
	assert.Equal(t, "adv-1-des", (templateAdvisories)["RH-1"].Advisory.Description)
	assert.Equal(t, "RH-2", (templateAdvisories)["RH-2"].Advisory.Name)
	assert.Equal(t, "adv-2-des", (templateAdvisories)["RH-2"].Advisory.Description)
}

func TestSyncTemplateAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	setupContentSourcesClient(t)

	templateID := int64(1)
	templateUUID := "99900000-0000-0000-0000-000000000001"
	orgID := "org_1"

	// DB has RH-1 and RH-2 linked to the template
	database.CreateTemplateAdvisories(t, accountID, templateID, []int64{1, 2})
	defer database.DeleteTemplateAdvisories(t, templateID, []int64{1, 2, 3})

	// content sources mock returns RH-1 and RH-3 so we need to add RH-3, remove RH-2
	ctx := identity.WithIdentity(context.Background(), utils.XRHIDForOrg(orgID))
	err := syncTemplateAdvisories(ctx, accountID, templateID, templateUUID)
	assert.Nil(t, err)

	// RH-3 and RH-1 now linked to the template
	database.CheckTemplateAdvisories(t, templateID, []int64{1, 3})

	// RH-2 was removed
	var count int64
	err = database.DB.Model(&models.TemplateAdvisory{}).
		Where("template_id = ? AND advisory_id = ?", templateID, 2).
		Count(&count).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(0), count)
}
