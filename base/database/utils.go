package database

import "github.com/jinzhu/gorm"

func SystemAdvisoriesQueryName(inventoryID string) *gorm.DB {
	query := systemAdvisoriesQuery().Where("sp.inventory_id = ?", inventoryID)
	return query
}

func SystemAdvisoriesQueryByID(systemID int) *gorm.DB {
	query := systemAdvisoriesQuery().Where("sp.id = ?", systemID)
	return query
}

func systemAdvisoriesQuery() *gorm.DB {
	query := Db.Table("system_advisories sa").Select("sa.*").
		Joins("join advisory_metadata am ON am.id=sa.advisory_id").
		Joins("join system_platform sp ON sa.system_id=sp.id")
	return query
}
