package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// nolint: unparam
func createTempateMsg(t *testing.T, eventName, orgID string, nTemplates int) mqueue.TemplateEvent {
	newUUID, err := uuid.NewRandom()
	assert.Nil(t, err)

	templates := make([]mqueue.TemplateResponse, nTemplates)
	for i := 0; i < nTemplates; i++ {
		description := fmt.Sprintf("Template%d description", i)
		templates[i] = mqueue.TemplateResponse{
			UUID:        fmt.Sprintf("77777777-0000-0000-0000-00000000000%1d", i),
			Name:        fmt.Sprintf("Template%d", i),
			OrgID:       orgID,
			Description: &description,
			Date:        time.Now(),
		}
	}

	event := mqueue.TemplateEvent{
		ID:      newUUID.String(),
		Source:  "urn:redhat:source:console:app:repositories",
		Type:    "com.redhat.console.repositories." + eventName,
		Subject: "urn:redhat:subject:console:rhel:" + eventName,
		Time:    time.Now(),
		OrgID:   orgID,
		Data:    templates,
	}

	return event
}

func testTemplatesInDB(t *testing.T) []models.Template {
	var templates []models.Template
	tx := database.DB.Model(&models.Template{}).
		Where("uuid::text like '77777777-%'").
		Order("uuid").
		Find(&templates)
	assert.Nil(t, tx.Error)
	return templates
}

func deleteTemplatesInDB(t *testing.T, templates []models.Template) {
	tx := database.DB.Delete(&templates)
	assert.Nil(t, tx.Error)
}

func TestCreateTemplate(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	createEvent := createTempateMsg(t, "template-created", "org_1", 3)
	msg, err := json.Marshal(createEvent)
	assert.Nil(t, err)

	// assert templates don't exist
	noTemplates := testTemplatesInDB(t)
	assert.Equal(t, 0, len(noTemplates))

	// process message
	err = TemplatesMessageHandler(mqueue.KafkaMessage{Value: msg})
	assert.Nil(t, err)

	// assert templates exist and have correct data
	postCreatedTemplates := testTemplatesInDB(t)
	assert.Equal(t, 3, len(postCreatedTemplates))
	assert.Equal(t, "77777777-0000-0000-0000-000000000000", postCreatedTemplates[0].UUID)
	assert.Equal(t, 1, postCreatedTemplates[0].RhAccountID)
	assert.Equal(t, "Template1", postCreatedTemplates[1].Name)
	assert.Equal(t, "Template2 description", *postCreatedTemplates[2].Description)

	updateEvent := createTempateMsg(t, "template-updated", "org_1", 2)
	updateEvent.Data[0].Name = "Updated Template0"
	description := "Updated Template1 description"
	updateEvent.Data[1].Description = &description
	msg, err = json.Marshal(updateEvent)
	assert.Nil(t, err)

	// process update
	err = TemplatesMessageHandler(mqueue.KafkaMessage{Value: msg})
	assert.Nil(t, err)

	// assert templates exist and have correct data
	postUpdatedTemplates := testTemplatesInDB(t)
	assert.Equal(t, 3, len(postUpdatedTemplates))
	assert.Equal(t, "Updated Template0", postUpdatedTemplates[0].Name)
	assert.Equal(t, "Updated Template1 description", *postUpdatedTemplates[1].Description)

	deleteEvent := createTempateMsg(t, "template-deleted", "org_1", 1)
	msg, err = json.Marshal(deleteEvent)
	assert.Nil(t, err)

	// process update
	err = TemplatesMessageHandler(mqueue.KafkaMessage{Value: msg})
	assert.Nil(t, err)

	// assert templates exist and have correct data
	postDeletedTemplates := testTemplatesInDB(t)
	assert.Equal(t, 2, len(postDeletedTemplates))
	assert.Equal(t, "77777777-0000-0000-0000-000000000001", postDeletedTemplates[0].UUID)
	assert.Equal(t, "77777777-0000-0000-0000-000000000002", postDeletedTemplates[1].UUID)

	// cleanup
	deleteTemplatesInDB(t, postDeletedTemplates)
}

func TestTemplateErrors(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	event := createTempateMsg(t, "template-created", "org_1", 3)
	event.Data[0].UUID = "not-an-uuid"
	event.Data[1].UUID = "neither-an-uuid"
	msg, err := json.Marshal(event)
	assert.Nil(t, err)

	// process message
	err = TemplatesMessageHandler(mqueue.KafkaMessage{Value: msg})
	expectedErr := errors.New(`creating template from message: ` +
		`ERROR: invalid input syntax for type uuid: "not-an-uuid" (SQLSTATE 22P02)\n` +
		`creating template from message: ` +
		`ERROR: invalid input syntax for type uuid: "neither-an-uuid" (SQLSTATE 22P02)`)
	assert.Error(t, expectedErr, err)

	// cleanup
	after := testTemplatesInDB(t)
	assert.Equal(t, 1, len(after))
	deleteTemplatesInDB(t, after)
}

func TestTemplateEmptyDescription(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	event := createTempateMsg(t, "template-created", "org_1", 3)
	empty := ""
	spaces := "   "
	event.Data[0].Description = &empty
	event.Data[1].Description = &spaces
	event.Data[2].Description = nil
	msg, err := json.Marshal(event)
	assert.Nil(t, err)

	// process message
	err = TemplatesMessageHandler(mqueue.KafkaMessage{Value: msg})
	assert.Nil(t, err)

	after := testTemplatesInDB(t)
	assert.Equal(t, 3, len(after))
	for _, event := range after {
		assert.Nil(t, event.Description)
	}

	// cleanup
	deleteTemplatesInDB(t, after)
}
