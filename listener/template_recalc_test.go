package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendTemplateRecalcDisabled(t *testing.T) {
	originalFlag := enableTemplateAdvisoryEval
	originalWriter := createdSystemsWriter
	defer func() {
		enableTemplateAdvisoryEval = originalFlag
		createdSystemsWriter = originalWriter
	}()

	mockWriter := &mqueue.MockKafkaWriter{}
	createdSystemsWriter = mockWriter
	enableTemplateAdvisoryEval = false

	err := SendTemplateRecalc(1, "org_1", 42)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(mockWriter.Messages))
}

func TestSendTemplateRecalcSkipsEmptyOrgID(t *testing.T) {
	originalFlag := enableTemplateAdvisoryEval
	originalWriter := createdSystemsWriter
	defer func() {
		enableTemplateAdvisoryEval = originalFlag
		createdSystemsWriter = originalWriter
	}()

	mockWriter := &mqueue.MockKafkaWriter{}
	createdSystemsWriter = mockWriter
	enableTemplateAdvisoryEval = true

	err := SendTemplateRecalc(1, "", 42)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(mockWriter.Messages))
}

func TestSendTemplateRecalcWithAssignedSystems(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	originalFlag := enableTemplateAdvisoryEval
	originalWriter := createdSystemsWriter
	defer func() {
		enableTemplateAdvisoryEval = originalFlag
		createdSystemsWriter = originalWriter
	}()

	mockWriter := &mqueue.MockKafkaWriter{}
	createdSystemsWriter = mockWriter
	enableTemplateAdvisoryEval = true

	const accountID = 1
	const orgID = "org_1"
	templateUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	inv1 := uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	inv2 := uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	createTestSystemInDB(t, inv1, accountID, "template-recalc-test-1")
	createTestSystemInDB(t, inv2, accountID, "template-recalc-test-2")
	t.Cleanup(func() {
		database.DeleteTemplate(t, accountID, templateUUID)
		deleteTestSystemInDB(t, inv1)
		deleteTestSystemInDB(t, inv2)
	})

	database.CreateTemplate(t, accountID, templateUUID, []uuid.UUID{inv1, inv2})

	var template models.Template
	err := database.DB.Where("rh_account_id = ? AND uuid = ?::uuid", accountID, templateUUID).First(&template).Error
	require.Nil(t, err)

	err = SendTemplateRecalc(accountID, orgID, template.ID)
	assert.Nil(t, err)
	require.Equal(t, 1, len(mockWriter.Messages))

	var event mqueue.PlatformEvent
	assert.Nil(t, sonic.Unmarshal(mockWriter.Messages[0].Value, &event))
	assert.Equal(t, orgID, event.GetOrgID())
	assert.Equal(t, accountID, event.AccountID)
	assert.ElementsMatch(t, []uuid.UUID{inv1, inv2}, event.SystemIDs)
}

func TestSendTemplateSystemsRecalc(t *testing.T) {
	originalFlag := enableTemplateAdvisoryEval
	originalWriter := createdSystemsWriter
	defer func() {
		enableTemplateAdvisoryEval = originalFlag
		createdSystemsWriter = originalWriter
	}()

	mockWriter := &mqueue.MockKafkaWriter{}
	createdSystemsWriter = mockWriter
	enableTemplateAdvisoryEval = true

	inv1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	inv2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	SendTemplateSystemsRecalc(1, "org_1", []uuid.UUID{inv1, inv2})
	assert.Equal(t, 1, len(mockWriter.Messages))

	var event mqueue.PlatformEvent
	assert.Nil(t, sonic.Unmarshal(mockWriter.Messages[0].Value, &event))
	assert.Equal(t, "org_1", event.GetOrgID())
	assert.Equal(t, 1, event.AccountID)
	assert.Equal(t, []uuid.UUID{inv1, inv2}, event.SystemIDs)

	mockWriter.Messages = nil
	SendTemplateSystemsRecalc(1, "org_1", nil)
	assert.Equal(t, 0, len(mockWriter.Messages))
}

func TestInventoryIDsToEvalData(t *testing.T) {
	inv := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	evalData := inventoryIDsToEvalData(1, "org_1", []uuid.UUID{inv})
	assert.Equal(t, 1, len(evalData))
	assert.Equal(t, inv, evalData[0].InventoryID)
	assert.Equal(t, 1, evalData[0].RhAccountID)
	assert.Equal(t, "org_1", *evalData[0].OrgID)
}
