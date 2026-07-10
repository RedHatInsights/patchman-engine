package listener

import (
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"app/manager/middlewares"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/pkg/errors"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
)

const (
	TemplateEventDelete = "template-deleted"
	TemplateEventUpdate = "template-updated"
	TemplateEventCreate = "template-created"
)

func TemplatesMessageHandler(m mqueue.KafkaMessage) error {
	eType, event, err := processTemplateEvent(m.Value)
	if err != nil {
		utils.LogError("err", err, "skipping template event")
		// Skip invalid messages
		return nil
	}

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
			err = TemplateUpdate(template, eType)
		default:
			utils.LogWarn("msg", fmt.Sprintf("%v", template), WarnUnknownType)
			err = nil
		}
		if err != nil {
			utils.LogError("err", err, "template", fmt.Sprintf("%v", template))
		}
	}
	return nil
}

func TemplateDelete(template mqueue.TemplateResponse) error {
	defer utils.ObserveSecondsSince(time.Now(), templateMsgHandlingDuration.WithLabelValues(TemplateEventDelete))

	// check account
	accountID, err := middlewares.GetOrCreateAccount(template.OrgID)
	if err != nil {
		return errors.Wrap(err, "when getting account")
	}

	var templateID int64
	err = database.DB.Model(&models.Template{}).
		Select("id").
		Where("rh_account_id = ? AND uuid = ?::uuid ", accountID, template.UUID).
		// use Find() not First() otherwise it returns error "no rows found" if uuid is not present
		Find(&templateID).Error
	if err != nil {
		return errors.Wrap(err, "getting template_id")
	}
	if templateID == 0 {
		utils.LogWarn("template", template, "template not found")
		return nil
	}

	err = SendTemplateRecalc(accountID, template.OrgID, templateID)
	if err != nil {
		return errors.Wrap(err, "sending template recalc")
	}

	// unassign systems from the template (template_id lives on system_patch)
	err = database.DB.Model(&models.SystemPatch{}).
		Where("rh_account_id = ? AND template_id = ?", accountID, templateID).
		Update("template_id", nil).Error
	if err != nil {
		return errors.Wrap(err, "removing systems from template")
	}

	err = database.DB.
		Delete(&models.TemplateAdvisory{}, "template_id = ? AND rh_account_id = ?", templateID, accountID).Error
	if err != nil {
		return errors.Wrap(err, "deleting template advisory relationships")
	}

	err = database.DB.
		Delete(&models.Template{}, "id = ? AND rh_account_id = ?", templateID, accountID).Error
	if err != nil {
		return errors.Wrap(err, "deleting template")
	}

	return nil
}

func TemplateUpdate(template mqueue.TemplateResponse, eventType string) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, templateMsgHandlingDuration.WithLabelValues(eventType))

	if template.OrgID == "" {
		utils.LogError("template", template.UUID, ErrorNoAccountProvided)
		eventMsgsReceivedCnt.WithLabelValues(eventType, ReceivedErrorIdentity).Inc()
		utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("template-skip"))
		return nil
	}

	// Ensure we have account stored
	accountID, err := middlewares.GetOrCreateAccount(template.OrgID)
	if err != nil {
		return errors.Wrap(err, "saving account into the database")
	}

	// fix empty EnvironmentID until Content Sources will start sending it
	if template.EnvironmentID == "" {
		template.EnvironmentID = strings.ReplaceAll(template.UUID, "-", "")
	}

	row := models.Template{
		TemplateBase: models.TemplateBase{
			RhAccountID: accountID,
			UUID:        template.UUID,
			Name:        template.Name,
		},
		EnvironmentID: template.EnvironmentID,
		Arch:          template.Arch,
		Version:       template.Version,
		//Config:      nil,
		Description: template.Description,
		Creator:     nil,
		Published:   &template.Date,
	}

	err = database.OnConflictUpdateMulti(database.DB, []string{"rh_account_id", "uuid"},
		"name", "environment_id", "description", "creator", "published").Save(&row).Error
	if err != nil {
		return errors.Wrap(err, "creating/updating template from message")
	}

	if contentSourcesBaseURL != "" && utils.CoreCfg.ContentSourcesUser != "" && eventType == TemplateEventUpdate {
		ctx := identity.WithIdentity(context.Background(),
			utils.XRHIDForOrg(template.OrgID, utils.CoreCfg.ContentSourcesUser))
		err = syncTemplateAdvisories(ctx, accountID, row.ID, template.UUID, template.OrgID)
		if err != nil {
			return errors.Wrap(err, "syncing template advisories")
		}
	}
	return nil
}

func processTemplateEvent(value json.RawMessage) (eType string, event mqueue.TemplateEvent, err error) {
	utils.LogTrace("kafka message data", string(value))
	if err := sonic.Unmarshal(value, &event); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("value: %s", string(value)))
		return "", event, errors.Wrap(err, "message is not a valid JSON")
	}

	for i, d := range event.Data {
		if d.Description != nil && strings.TrimSpace(*d.Description) == "" {
			d.Description = nil
			event.Data[i] = d
		}
	}
	return strings.TrimPrefix(event.Type, "com.redhat.console.repositories."), event, nil
}
