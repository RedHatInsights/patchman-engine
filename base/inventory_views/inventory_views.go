package inventory_views

import (
	"app/base/models"
	"app/base/utils"
	"time"

	"gorm.io/gorm"
)

type InventoryViewsHost struct {
	// Inventory ID (UUID) of the host
	ID   string                 `json:"id"`
	Data InventoryViewsHostData `json:"data"`
}

type InventoryViewsHostData struct {
	ApplicableRhsaCount   int     `json:"advisories_rhsa_applicable"`
	ApplicableRhbaCount   int     `json:"advisories_rhba_applicable"`
	ApplicableRheaCount   int     `json:"advisories_rhea_applicable"`
	ApplicableOtherCount  int     `json:"advisories_other_applicable"`
	InstallableRhsaCount  int     `json:"advisories_rhsa_installable"`
	InstallableRhbaCount  int     `json:"advisories_rhba_installable"`
	InstallableRheaCount  int     `json:"advisories_rhea_installable"`
	InstallableOtherCount int     `json:"advisories_other_installable"`
	PackagesInstalled     int     `json:"packages_installed"`
	PackagesInstallable   int     `json:"packages_installable"`
	PackagesApplicable    int     `json:"packages_applicable"`
	TemplateName          *string `json:"template_name"`
	TemplateUUID          *string `json:"template_uuid"`
}

type InventoryViewsEvent struct {
	OrgID     string               `json:"org_id"`
	Timestamp string               `json:"timestamp"`
	Hosts     []InventoryViewsHost `json:"hosts"`
}

func MakeInventoryViewsEvent(tx *gorm.DB, orgID string, systems []models.SystemPlatform) (
	InventoryViewsEvent, error) {
	templates, err := FindSystemsTemplates(tx, systems)
	if err != nil {
		return InventoryViewsEvent{}, err
	}
	hosts := MakeInventoryViewsHosts(systems, templates)
	return InventoryViewsEvent{OrgID: orgID, Timestamp: time.Now().Format(time.RFC3339), Hosts: hosts}, nil
}

func MakeInventoryViewsHosts(systems []models.SystemPlatform,
	templates map[int64]models.TemplateBase) []InventoryViewsHost {
	hosts := make([]InventoryViewsHost, len(systems))
	for i, system := range systems {
		hosts[i] = InventoryViewsHost{
			ID: system.InventoryID,
			Data: InventoryViewsHostData{
				ApplicableRhsaCount: system.ApplicableAdvisorySecCountCache,
				ApplicableRhbaCount: system.ApplicableAdvisoryBugCountCache,
				ApplicableRheaCount: system.ApplicableAdvisoryEnhCountCache,
				ApplicableOtherCount: system.ApplicableAdvisoryCountCache - system.ApplicableAdvisorySecCountCache -
					system.ApplicableAdvisoryBugCountCache - system.ApplicableAdvisoryEnhCountCache,
				InstallableRhsaCount: system.InstallableAdvisorySecCountCache,
				InstallableRhbaCount: system.InstallableAdvisoryBugCountCache,
				InstallableRheaCount: system.InstallableAdvisoryEnhCountCache,
				InstallableOtherCount: system.InstallableAdvisoryCountCache - system.InstallableAdvisorySecCountCache -
					system.InstallableAdvisoryBugCountCache - system.InstallableAdvisoryEnhCountCache,
				PackagesInstalled:   system.PackagesInstalled,
				PackagesInstallable: system.PackagesInstallable,
				PackagesApplicable:  system.PackagesApplicable,
			},
		}
		if system.TemplateID != nil {
			template, ok := templates[*system.TemplateID]
			if ok {
				hosts[i].Data.TemplateName = &template.Name
				hosts[i].Data.TemplateUUID = &template.UUID
			} else {
				utils.LogWarn("template_id", system.TemplateID, "template not found")
			}
		}
	}
	return hosts
}

func FindSystemsTemplates(tx *gorm.DB, systems []models.SystemPlatform) (map[int64]models.TemplateBase, error) {
	templateIDs := make([]int64, 0, len(systems))
	if len(systems) == 0 {
		return nil, nil
	}
	for _, system := range systems {
		if system.TemplateID == nil {
			continue
		}
		templateIDs = append(templateIDs, *system.TemplateID)
	}

	if len(templateIDs) == 0 {
		return nil, nil
	}
	templates := make([]models.TemplateBase, 0, len(templateIDs))
	q := tx.Model(&models.TemplateBase{}).
		Where("rh_account_id = ? AND id IN (?)", systems[0].RhAccountID, templateIDs)
	err := q.Find(&templates).Error
	if err != nil {
		return nil, err
	}

	templatesMap := make(map[int64]models.TemplateBase, len(templates))
	for _, t := range templates {
		templatesMap[t.ID] = t
	}
	return templatesMap, nil
}
