package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"app/listener"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareTemplateAdvisoryIntegrationTest() {
	configure()
	enableTemplateAdvisoryEval = true
	enableVmaasCache = false
	loadCache()
	remediationsPublisher = &mqueue.MockKafkaWriter{}
}

func evaluateHandlerForTest(event mqueue.PlatformEvent) error {
	data, err := sonic.Marshal(event)
	if err != nil {
		return err
	}
	return evaluateHandler(mqueue.KafkaMessage{Value: data})
}

// nolint: funlen
func TestTemplateAdvisoryEvalE2E(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()

	mockWriter, cleanupListener := listener.InitForTemplateAdvisoryIntegrationTest(t)
	defer cleanupListener()

	const accountID = 1
	const orgID = "org_1"
	templateUUID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	inventoryID := uuid.MustParse("00000000-0000-0000-0000-0000000000e1")
	vmaasJSON := `{ "package_list": [ "firefox-0:76.0.1-1.fc31.x86_64", "kernel-0:5.6.13-200.fc31.x86_64" ], "repository_list": [ "repo1" ] }` //nolint:lll

	systemInv := createE2ETestSystem(t, inventoryID, accountID, vmaasJSON)
	database.CreateSystemRepos(t, accountID, systemInv.ID, []int64{1})
	t.Cleanup(func() {
		database.DeleteSystemRepos(t, accountID, systemInv.ID, []int64{1})
		database.DeleteTemplate(t, accountID, templateUUID)
		deleteE2ETestSystem(t, inventoryID)
	})

	database.CreateTemplate(t, accountID, templateUUID, []uuid.UUID{inventoryID})

	var template models.Template
	err := database.DB.Where("rh_account_id = ? AND uuid = ?::uuid", accountID, templateUUID).First(&template).Error
	require.NoError(t, err)

	database.CreateTemplateAdvisories(t, accountID, template.ID, []int64{1, 2})
	defer database.DeleteTemplateAdvisories(t, template.ID, []int64{1, 2, 3})

	database.DeleteSystemAdvisories(t, systemInv.ID, []int64{1, 2, 3, 100})
	database.DeleteAdvisoryAccountData(t, accountID, []int64{1, 2, 3, 100})

	description := "e2e template"
	updateEvent := mqueue.TemplateEvent{
		Type:  "com.redhat.console.repositories." + listener.TemplateEventUpdate,
		OrgID: orgID,
		Data: []mqueue.TemplateResponse{{
			UUID:          templateUUID,
			EnvironmentID: strings.ReplaceAll(templateUUID, "-", ""),
			Name:          "e2e-template",
			OrgID:         orgID,
			Description:   &description,
			Date:          time.Now(),
			Arch:          "x86_64",
			Version:       "8",
		}},
	}
	msg, err := sonic.Marshal(updateEvent)
	require.NoError(t, err)

	err = listener.TemplatesMessageHandler(mqueue.KafkaMessage{Value: msg})
	require.NoError(t, err)

	database.CheckTemplateAdvisories(t, template.ID, []int64{1, 3})
	require.Equal(t, 1, len(mockWriter.Messages))

	var recalcEvent mqueue.PlatformEvent
	require.NoError(t, sonic.Unmarshal(mockWriter.Messages[0].Value, &recalcEvent))
	require.Equal(t, accountID, recalcEvent.AccountID)
	require.Equal(t, orgID, recalcEvent.GetOrgID())
	require.Equal(t, []uuid.UUID{inventoryID}, recalcEvent.SystemIDs)

	prepareTemplateAdvisoryIntegrationTest()
	err = evaluateHandlerForTest(recalcEvent)
	require.NoError(t, err)

	assertSystemAdvisoryStatus(t, systemInv.ID, "RH-1", INSTALLABLE)
	assertSystemAdvisoryStatus(t, systemInv.ID, "RH-2", APPLICABLE)
	assertSystemAdvisoryStatus(t, systemInv.ID, "RH-100", APPLICABLE)
}

func createE2ETestSystem(t *testing.T, inventoryID uuid.UUID, accountID int, vmaasJSON string) models.SystemInventory {
	t.Helper()
	inv := models.SystemInventory{
		InventoryID:  inventoryID,
		RhAccountID:  accountID,
		DisplayName:  "template-advisory-e2e",
		Tags:         []byte("[]"),
		VmaasJSON:    &vmaasJSON,
		JSONChecksum: strPtr("e2e-template-advisory"),
	}
	require.NoError(t, database.DB.Create(&inv).Error)
	require.NoError(t, database.DB.Create(&models.SystemPatch{
		SystemID:    inv.ID,
		RhAccountID: accountID,
	}).Error)
	return inv
}

func deleteE2ETestSystem(t *testing.T, inventoryID uuid.UUID) {
	t.Helper()
	var inv models.SystemInventory
	err := database.DB.Where("inventory_id = ?", inventoryID).First(&inv).Error
	if err != nil {
		return
	}
	require.NoError(t, database.DB.Unscoped().
		Where("rh_account_id = ? AND system_id = ?", inv.RhAccountID, inv.ID).
		Delete(&models.SystemAdvisories{}).Error)
	require.NoError(t, database.DB.Unscoped().
		Where("rh_account_id = ? AND system_id = ?", inv.RhAccountID, inv.ID).
		Delete(&models.SystemPackage{}).Error)
	require.NoError(t, database.DB.Unscoped().
		Where("rh_account_id = ? AND system_id = ?", inv.RhAccountID, inv.ID).
		Delete(&models.SystemRepo{}).Error)
	require.NoError(t, database.DB.Unscoped().Exec(
		"DELETE FROM system_patch WHERE rh_account_id = ? AND system_id = ?",
		inv.RhAccountID, inv.ID).Error)
	require.NoError(t, database.DB.Unscoped().Where("inventory_id = ?", inventoryID).
		Delete(&models.SystemInventory{}).Error)
}

func assertSystemAdvisoryStatus(t *testing.T, systemID int64, advisoryName string, expectedStatus int) {
	t.Helper()
	var systemAdvisory models.SystemAdvisories
	err := database.DB.Table("system_advisories sa").
		Select("sa.*").
		Joins("JOIN advisory_metadata am ON am.id = sa.advisory_id").
		Where("sa.system_id = ? AND am.name = ?", systemID, advisoryName).
		First(&systemAdvisory).Error
	require.NoError(t, err)
	assert.Equal(t, expectedStatus, systemAdvisory.StatusID)
}

func strPtr(s string) *string {
	return &s
}
