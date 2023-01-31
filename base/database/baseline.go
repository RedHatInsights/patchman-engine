package database

import (
	"app/base/models"
	"app/base/utils"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type BaselineConfig struct {
	// Filter applicable advisories (updates) by the latest publish time.
	ToTime time.Time `json:"to_time" example:"2022-12-31T12:00:00-04:00"`
}

func GetBaselineConfig(tx *gorm.DB, system *models.SystemPlatform) *BaselineConfig {
	if system.BaselineID == nil {
		return nil
	}

	var jsonb [][]byte
	err := tx.Table("baseline").
		Where("id = ? and rh_account_id = ?", system.BaselineID, system.RhAccountID).
		Pluck("config", &jsonb).Error
	if err != nil {
		utils.LogError("baseline_id", system.BaselineID, "err", err.Error(),
			"Unable to load baseline config from db")
		return nil
	}

	var config BaselineConfig
	if len(jsonb[0]) == 0 {
		utils.LogDebug("baseline_id", system.BaselineID, "Empty baseline config found")
		return nil
	}

	err = json.Unmarshal(jsonb[0], &config)
	if err != nil {
		utils.LogError("err", err.Error(), "baseline_id", system.BaselineID,
			"config", string(jsonb[0]), "Can't parse baseline")
		return nil
	}
	return &config
}
