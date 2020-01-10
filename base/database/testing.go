package database

import (
	"app/base/models"
	"github.com/stretchr/testify/assert"
	"testing"
)

func CheckAdvisoriesInDb(t *testing.T, advisories []string) []int {
	var advisoriesObjs []models.AdvisoryMetadata
	err := Db.Where("name IN (?)", advisories).Find(&advisoriesObjs).Error
	assert.Nil(t, err)
	assert.Equal(t, len(advisoriesObjs), len(advisories))
	var ids []int
	for _, advisoryObj := range advisoriesObjs {
		ids = append(ids, advisoryObj.ID)
	}
	return ids
}

func CheckSystemAdvisoriesFirstReportedGreater(t *testing.T, firstReported string, count int) {
	var systemAdvisories []models.SystemAdvisories
	err := Db.Where("first_reported > ?", firstReported).
		Find(&systemAdvisories).Error
	assert.Nil(t, err)
	assert.Equal(t, count, len(systemAdvisories))
}
