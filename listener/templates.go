package listener

import (
	"app/base"
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
	"gorm.io/gorm"
)

const (
	TemplateEventDelete = "template-deleted"
	TemplateEventUpdate = "template-updated"
	TemplateEventCreate = "template-created"
)

func TemplatesMessageHandler(ctx context.Context, m mqueue.KafkaMessage) error {
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
			err = TemplateDelete(ctx, template)
		case TemplateEventUpdate:
			fallthrough
		case TemplateEventCreate:
			err = TemplateUpdate(ctx, template, eType)
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

func TemplateDelete(ctx context.Context, template mqueue.TemplateResponse) error {
	defer utils.ObserveSecondsSince(time.Now(), templateMsgHandlingDuration.WithLabelValues(TemplateEventDelete))

	// check account
	accountID, err := middlewares.GetOrCreateAccount(ctx, template.OrgID)
	if err != nil {
		return errors.Wrap(err, "when getting account")
	}

	var templateID int64
	err = database.DB(ctx).Model(&models.Template{}).
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

	err = SendTemplateRecalc(ctx, accountID, template.OrgID, templateID)
	if err != nil {
		return errors.Wrap(err, "sending template recalc")
	}

	// unassign systems from the template (template_id lives on system_patch)
	err = database.DB(ctx).Model(&models.SystemPatch{}).
		Where("rh_account_id = ? AND template_id = ?", accountID, templateID).
		Update("template_id", nil).Error
	if err != nil {
		return errors.Wrap(err, "removing systems from template")
	}

	err = database.DB(ctx).
		Delete(&models.TemplateAdvisory{}, "template_id = ? AND rh_account_id = ?", templateID, accountID).Error
	if err != nil {
		return errors.Wrap(err, "deleting template advisory relationships")
	}

	err = database.DB(ctx).
		Delete(&models.Template{}, "id = ? AND rh_account_id = ?", templateID, accountID).Error
	if err != nil {
		return errors.Wrap(err, "deleting template")
	}

	return nil
}

func TemplateUpdate(ctx context.Context, template mqueue.TemplateResponse, eventType string) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, templateMsgHandlingDuration.WithLabelValues(eventType))

	if template.OrgID == "" {
		utils.LogError("template", template.UUID, ErrorNoAccountProvided)
		eventMsgsReceivedCnt.WithLabelValues(eventType, ReceivedErrorIdentity).Inc()
		utils.ObserveSecondsSince(tStart, messagePartDuration.WithLabelValues("template-skip"))
		return nil
	}

	// Ensure we have account stored
	accountID, err := middlewares.GetOrCreateAccount(ctx, template.OrgID)
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

	err = saveTemplate(ctx, &row)
	if err != nil {
		return err
	}

	switch {
	case templatesWithAdvisories:
		err = applyTemplateAdvisories(ctx, accountID, row.ID, template.UUID, template.OrgID, template.AdvisoryIDs)
		if err != nil {
			return errors.Wrap(err, "syncing template advisories from message")
		}
	case contentSourcesBaseURL != "" && utils.CoreCfg.ContentSourcesUser != "" && eventType == TemplateEventUpdate:
		csCtx := identity.WithIdentity(base.Context,
			utils.XRHIDForOrg(template.OrgID, utils.CoreCfg.ContentSourcesUser))
		err = syncTemplateAdvisories(csCtx, accountID, row.ID, template.UUID, template.OrgID)
		if err != nil {
			return errors.Wrap(err, "syncing template advisories")
		}
	}
	return nil
}

// saveTemplate updates an existing template or inserts new one
// don't use INSERT ... ON CONFLICT, it calls nextval('template_id_seq') before the
// conflict check and advances the sequence even if the row already exists
func saveTemplate(ctx context.Context, row *models.Template) error {
	db := database.DB(ctx)

	templateID, err := findTemplateID(db, row.RhAccountID, row.UUID)
	if err != nil {
		return err
	}
	if templateID != 0 {
		row.ID = templateID
		return updateTemplate(db, row)
	}

	err = insertTemplate(db, row)
	if err != nil && database.IsPgErrorCode(db, err, gorm.ErrDuplicatedKey) {
		templateID, err = findTemplateID(db, row.RhAccountID, row.UUID)
		if err != nil {
			return err
		}
		if templateID == 0 {
			return errors.New("template missing after unique conflict")
		}
		row.ID = templateID
		return updateTemplate(db, row)
	}
	return err
}

func findTemplateID(db *gorm.DB, accountID int, templateUUID string) (int64, error) {
	var templateID int64
	err := db.Model(&models.Template{}).
		Select("id").
		Where("rh_account_id = ? AND uuid = ?::uuid", accountID, templateUUID).
		Find(&templateID).Error
	if err != nil {
		return 0, errors.Wrap(err, "getting template")
	}
	return templateID, nil
}

func updateTemplate(db *gorm.DB, row *models.Template) error {
	err := db.Model(&models.Template{}).
		Where("rh_account_id = ? AND id = ?", row.RhAccountID, row.ID).
		Updates(map[string]interface{}{
			"name":           row.Name,
			"environment_id": row.EnvironmentID,
			"description":    row.Description,
			"creator":        row.Creator,
			"published":      row.Published,
		}).Error
	if err != nil {
		return errors.Wrap(err, "updating template from message")
	}
	return nil
}

func insertTemplate(db *gorm.DB, row *models.Template) error {
	err := db.Omit("ID").Create(row).Error
	if err != nil {
		return errors.Wrap(err, "creating template from message")
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
