package listener

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
)

const orgID = "org_1"

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
	nameToID, err := lookUpAdvisoryMetadataIDs(base.Context, advisoryNames)
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

	templateAdvisories, err := lookUpTemplateAdvisories(database.TestDB(), accountID, templateID)
	assert.Nil(t, err)
	assert.NotNil(t, templateAdvisories)
	assert.Equal(t, 2, len(templateAdvisories))
	assert.Equal(t, "RH-1", (templateAdvisories)["RH-1"].Advisory.Name)
	assert.Equal(t, "adv-1-des", (templateAdvisories)["RH-1"].Advisory.Description)
	assert.Equal(t, "RH-2", (templateAdvisories)["RH-2"].Advisory.Name)
	assert.Equal(t, "adv-2-des", (templateAdvisories)["RH-2"].Advisory.Description)
}

func disableTemplateAdvisoryEval(t *testing.T) {
	t.Helper()
	originalEval := enableTemplateAdvisoryEval
	enableTemplateAdvisoryEval = false
	t.Cleanup(func() {
		enableTemplateAdvisoryEval = originalEval
	})
}

func templateAdvisoryCount(t *testing.T, templateID int64) int64 {
	t.Helper()
	var count int64
	err := database.TestDB().Model(&models.TemplateAdvisory{}).
		Where("template_id = ?", templateID).
		Count(&count).Error
	assert.Nil(t, err)
	return count
}

func loadTemplateByUUID(t *testing.T, templateUUID string) models.Template {
	t.Helper()
	var stored models.Template
	err := database.TestDB().
		Where("rh_account_id = ? AND uuid = ?::uuid", accountID, templateUUID).
		First(&stored).Error
	assert.Nil(t, err)
	return stored
}

func TestTemplateMessageAdvisoryIDs(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	disableTemplateAdvisoryEval(t)

	templateUUID := "88888888-0000-0000-0000-000000000099"
	t.Cleanup(func() {
		database.DeleteTemplate(t, accountID, templateUUID)
	})

	createEvent := createTempateMsg(t, "template-created", orgID, 1)
	createEvent.Data[0].UUID = templateUUID
	createEvent.Data[0].EnvironmentID = "88888888000000000000000000000099"
	createEvent.Data[0].AdvisoryIDs = []string{"RH-1", "RH-2"}
	msg, err := sonic.Marshal(createEvent)
	assert.Nil(t, err)

	err = TemplatesMessageHandler(base.Context, mqueue.KafkaMessage{Value: msg})
	assert.Nil(t, err)

	stored := loadTemplateByUUID(t, templateUUID)
	database.CheckTemplateAdvisories(t, stored.ID, []int64{1, 2})
	assert.Equal(t, int64(2), templateAdvisoryCount(t, stored.ID))

	updateEvent := createEvent
	updateEvent.Type = "com.redhat.console.repositories.template-updated"
	updateEvent.Data[0].AdvisoryIDs = []string{"RH-1", "RH-3"}
	msg, err = sonic.Marshal(updateEvent)
	assert.Nil(t, err)

	err = TemplatesMessageHandler(base.Context, mqueue.KafkaMessage{Value: msg})
	assert.Nil(t, err)

	database.CheckTemplateAdvisories(t, stored.ID, []int64{1, 3})
	assert.Equal(t, int64(2), templateAdvisoryCount(t, stored.ID))
	var removed int64
	err = database.TestDB().Model(&models.TemplateAdvisory{}).
		Where("template_id = ? AND advisory_id = ?", stored.ID, 2).
		Count(&removed).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(0), removed)

	updateEvent.Data[0].AdvisoryIDs = []string{}
	msg, err = sonic.Marshal(updateEvent)
	assert.Nil(t, err)
	err = TemplatesMessageHandler(base.Context, mqueue.KafkaMessage{Value: msg})
	assert.Nil(t, err)
	assert.Equal(t, int64(0), templateAdvisoryCount(t, stored.ID))
}
