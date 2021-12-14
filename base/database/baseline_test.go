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
	baseline, err := GetBaselineConfig(Db, &system)
	assert.Nil(t, err)
	assert.Nil(t, baseline)

	// system with existing baseline
	system = models.SystemPlatform{ID: 1, RhAccountID: 1, BaselineID: utils.PtrInt(1)}
	baseline, err = GetBaselineConfig(Db, &system)
	assert.Nil(t, err)
	assert.Equal(t, "2010-09-22 00:00:00+00", baseline.ToTime.Format("2006-01-02 15:04:05-07"))
}
