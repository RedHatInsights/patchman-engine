package database

import (
	"app/base/models"
	"app/base/utils"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type BaselineConfig struct {
	ToTime time.Time `json:"to_time"`
}

func GetBaselineConfig(tx *gorm.DB, system *models.SystemPlatform) (*BaselineConfig, error) {
	if system.BaselineID == nil {
		return nil, nil
	}

	var config BaselineConfig
	var jsonb [][]byte

	err := tx.Table("baseline").
		Where("id = ? and rh_account_id = ?", system.BaselineID, system.RhAccountID).
		Pluck("config", &jsonb).Error
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonb[0], &config)
	if err != nil {
		utils.Log("err", err.Error(), "baseline", system.BaselineID).Error("Can't parse baseline")
		return nil, err
	}
	return &config, err
}
