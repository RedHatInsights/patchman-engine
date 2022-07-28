package database

import (
	"app/base/models"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaselineConfig(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	// system without baseline
	system := models.SystemPlatform{ID: 8, RhAccountID: 1, BaselineID: nil}
	baselineConfig := GetBaselineConfig(Db, &system)
	assert.Nil(t, baselineConfig)

	// system with existing baseline
	system = models.SystemPlatform{ID: 1, RhAccountID: 1, BaselineID: utils.PtrInt64(1)}
	baselineConfig = GetBaselineConfig(Db, &system)
	assert.Equal(t, "2010-09-22 00:00:00+00", baselineConfig.ToTime.Format("2006-01-02 15:04:05-07"))

	baselineID := CreateBaselineWithConfig(t, "", nil, nil)
	// baseline with empty config
	system = models.SystemPlatform{ID: 1, RhAccountID: 1, BaselineID: &baselineID}
	baselineConfig = GetBaselineConfig(Db, &system)
	assert.Nil(t, baselineConfig)
	DeleteBaseline(t, baselineID)
}
