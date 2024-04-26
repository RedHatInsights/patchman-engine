package listener

import (
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	errors2 "errors"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	TemplateEventDelete = "template-deleted"
	TemplateEventUpdate = "template-updated"
	TemplateEventCreate = "template-created"
)

func TemplatesMessageHandler(m mqueue.KafkaMessage) error {
	eType, event, err := processTemplateEvent(m.Value)
	if err != nil {
		utils.LogError("msg", err.Error(), "skipping template event")
		// Skip invalid messages
		return nil
	}

	errs := []error{}
	for _, template := range event.Data {
		if enableBypass {
			utils.LogInfo("template", template.UUID, "Processing bypassed")
			templateMsgReceivedCnt.WithLabelValues(eType, ReceivedBypassed).Inc()
			continue
		}

		switch eType {
		case TemplateEventDelete:
			err = TemplateDelete(template)
		case TemplateEventUpdate:
			fallthrough
		case TemplateEventCreate:
			err = TemplateUpdate(template)
		default:
			utils.LogWarn("msg", fmt.Sprintf("%v", template), WarnUnknownType)
			err = nil
		}
		errs = append(errs, err)
	}
	err = errors2.Join(errs...)
	// join errors and return
	return err
}

func TemplateDelete(template mqueue.TemplateResponse) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, templateMsgHandlingDuration.WithLabelValues(TemplateEventDelete))

	err := database.Db.
		Delete(&models.Template{}, "uuid = ?::uuid AND rh_account_id = (SELECT id FROM rh_account WHERE org_id = ?)",
			template.UUID, template.OrgID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		utils.LogWarn("template", template.UUID, WarnNoRowsModified)
		return nil
	}
	return err
}

func TemplateUpdate(template mqueue.TemplateResponse) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, templateMsgHandlingDuration.WithLabelValues(TemplateEventCreate))

	if template.OrgID == "" {
		utils.LogError("template", template.UUID, ErrorNoAccountProvided)
		eventMsgsReceivedCnt.WithLabelValues(TemplateEventCreate, ReceivedErrorIdentity).Inc()
		utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("template-skip"))
		return nil
	}

	// Ensure we have account stored
	accountID, err := middlewares.GetOrCreateAccount(template.OrgID)
	if err != nil {
		return errors.Wrap(err, "saving account into the database")
	}

	row := models.Template{
		RhAccountID: accountID,
		UUID:        template.UUID,
		Name:        template.Name,
		//Config:      nil,
		Description: template.Description,
		Creator:     nil,
		Published:   &template.Date,
	}

	err = database.OnConflictUpdateMulti(database.Db, []string{"rh_account_id", "uuid"},
		"name", "description", "creator", "published").Save(&row).Error
	if err != nil {
		return errors.Wrap(err, "creating template from message")
	}
	return nil
}

func processTemplateEvent(value json.RawMessage) (eType string, event mqueue.TemplateEvent, err error) {
	utils.LogTrace("kafka message data", string(value))
	if err := json.Unmarshal(value, &event); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("value: %s", string(value)))
		return "", event, errors.Wrap(err, "message is not a valid JSON")
	}

	for i, d := range event.Data {
		if d.Description != nil && (len(*d.Description) == 0 || spacesRegex.MatchString(*d.Description)) {
			d.Description = nil
			event.Data[i] = d
		}
	}
	return strings.TrimPrefix(event.Type, "com.redhat.console.repositories."), event, nil
}
