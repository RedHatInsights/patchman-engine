package evaluator

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func useTemplateAdvisoryEval(system *models.SystemPlatformV2) bool {
	return enableTemplateAdvisoryEval && system.Patch.TemplateID != nil &&
		!system.Inventory.SatelliteManaged
}

// getTemplateAdvisoryUpdatesData evaluates template-assigned systems using VMaaS only.
// Yum updates are intentionally not merged; installability comes from template_advisory.
func getTemplateAdvisoryUpdatesData(ctx context.Context, system *models.SystemPlatformV2) (
	*vmaas.UpdatesV3Response, error) {
	vmaasData, vmaasErr := getVmaasUpdates(ctx, system)
	if vmaasErr != nil {
		if errors.Is(vmaasErr, errVmaasBadRequest) {
			utils.LogWarn("Vmaas response error - bad request, skipping system", vmaasErr.Error())
			return nil, nil
		}
		return nil, errors.Wrap(vmaasErr, "getting vmaas updates for template-advisory evaluation")
	}
	if vmaasData == nil {
		return nil, nil
	}

	templateErrata, err := loadTemplateAdvisoryErrata(database.DB, system.Inventory.RhAccountID,
		*system.Patch.TemplateID)
	if err != nil {
		return nil, errors.Wrap(err, "loading template advisories")
	}
	applyTemplateAdvisoryInstallability(vmaasData, templateErrata)
	return vmaasData, nil
}

func loadTemplateAdvisoryErrata(tx *gorm.DB, accountID int, templateID int64) (map[string]struct{}, error) {
	var data []models.TemplateAdvisory
	err := tx.Preload("Advisory").
		Find(&data, "template_id = ? AND rh_account_id = ?", templateID, accountID).Error
	if err != nil {
		return nil, err
	}

	errata := make(map[string]struct{}, len(data))
	for _, ta := range data {
		errata[ta.Advisory.Name] = struct{}{}
	}
	return errata, nil
}

func applyTemplateAdvisoryInstallability(vmaasData *vmaas.UpdatesV3Response, templateErrata map[string]struct{}) {
	if vmaasData == nil {
		return
	}
	updateList := vmaasData.GetUpdateList()
	for _, updates := range updateList {
		if updates == nil || updates.AvailableUpdates == nil {
			continue
		}
		for i := range *updates.AvailableUpdates {
			u := &(*updates.AvailableUpdates)[i]
			erratum := u.GetErratum()
			if len(erratum) == 0 {
				continue
			}
			if _, inTemplate := templateErrata[erratum]; inTemplate {
				u.SetInstallability(INSTALLABLE)
			} else {
				u.SetInstallability(APPLICABLE)
			}
		}
	}
}
