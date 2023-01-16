package models

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

// test association SystemAdvisories.Advisory
func TestSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var systemAdvisories []models.SystemAdvisories
	err := database.Db.Model(models.SystemAdvisories{}).Preload("Advisory").
		Where("system_id = ?", 1).Find(&systemAdvisories).Error
	assert.Nil(t, err)
	assert.Equal(t, 8, len(systemAdvisories))
	assert.Equal(t, "RH-1", systemAdvisories[0].Advisory.Name)
}
