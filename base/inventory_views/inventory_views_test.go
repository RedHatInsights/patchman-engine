package inventory_views

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const rhAccountID = 1

func TestMakeInventoryViewsEvent(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	tx := database.DB.Begin()
	defer tx.Rollback()

	var rhAccount models.RhAccount
	assert.NoError(t, tx.Where("id = ?", rhAccountID).First(&rhAccount).Error)
	assert.NotEmpty(t, *rhAccount.OrgID)

	var systems []models.SystemPlatform
	assert.NoError(t, tx.Where("rh_account_id = ? AND id in (1,3)", rhAccountID).
		Order("id").Find(&systems).Error)
	assert.Equal(t, 2, len(systems))

	event, err := MakeInventoryViewsEvent(tx, *rhAccount.OrgID, systems)
	assert.NoError(t, err)
	assert.Equal(t, *rhAccount.OrgID, event.OrgID)
	assert.NotEmpty(t, event.Timestamp)
	_, err = time.Parse(time.RFC3339, event.Timestamp)
	assert.NoError(t, err)

	// Verify hosts
	assert.Equal(t, 2, len(event.Hosts))

	assert.Equal(t, InventoryViewsHost{
		ID: "00000000-0000-0000-0000-000000000001",
		Data: InventoryViewsHostData{2, 3, 3, 0, 2, 2, 1, 0, 0, 0, 0,
			utils.PtrString("temp1-1"), utils.PtrString("99900000-0000-0000-0000-000000000001")},
	}, event.Hosts[0])

	assert.Equal(t, InventoryViewsHost{
		ID: "00000000-0000-0000-0000-000000000003",
		Data: InventoryViewsHostData{0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0,
			utils.PtrString("temp2-1"), utils.PtrString("99900000-0000-0000-0000-000000000002")},
	}, event.Hosts[1])
}

func TestMakeInventoryViewsEventEmpty(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	tx := database.DB.Begin()
	defer tx.Rollback()

	event, err := MakeInventoryViewsEvent(tx, "test-org", []models.SystemPlatform{})
	assert.NoError(t, err)
	assert.Equal(t, "test-org", event.OrgID)
	assert.Equal(t, 0, len(event.Hosts))
	assert.NotEmpty(t, event.Timestamp)
}

func TestMakeInventoryViewsEventNoTemplate(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	tx := database.DB.Begin()
	defer tx.Rollback()

	var rhAccount models.RhAccount
	assert.NoError(t, tx.Where("id = ?", rhAccountID).First(&rhAccount).Error)
	assert.NotEmpty(t, *rhAccount.OrgID)
	orgID := *rhAccount.OrgID

	var systems []models.SystemPlatform
	assert.NoError(t, tx.Where("rh_account_id = ? AND id in (4)", rhAccountID).
		Order("id").Find(&systems).Error)
	assert.Equal(t, 1, len(systems))

	// Should not error, but template fields should be nil
	event, err := MakeInventoryViewsEvent(tx, orgID, systems)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(event.Hosts))

	host := event.Hosts[0]
	assert.Equal(t, "00000000-0000-0000-0000-000000000004", host.ID)
	// Template fields should be nil when template is not found
	assert.Nil(t, host.Data.TemplateName)
	assert.Nil(t, host.Data.TemplateUUID)
}
