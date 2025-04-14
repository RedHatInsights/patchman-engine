package listener

import (
	"app/base/models"
	"app/base/utils"

	"gorm.io/gorm"
)

func getTemplate(db *gorm.DB, accountID int, environments []string) (*int64, error) {
	var templateID *int64
	if len(environments) == 0 {
		// no environments
		return templateID, nil
	}

	// get template ids for given environments
	var environmentTemplates []int64
	err := db.Model(models.Template{}).
		Where("rh_account_id = ? AND environment_id IN (?)", accountID, environments).
		Select("id").
		Scan(&environmentTemplates).Error
	if err != nil {
		return nil, err
	}

	if len(environmentTemplates) == 0 {
		return templateID, nil
	}

	templateID = &environmentTemplates[0]
	if len(environmentTemplates) > 1 {
		utils.LogWarn(
			"account", accountID, "environments", environments, "templates", environmentTemplates,
			"Multiple templates found in account rhsm environments",
		)
	}
	return templateID, nil
}
