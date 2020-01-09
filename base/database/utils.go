package database

import "github.com/jinzhu/gorm"

func SystemAdvisoriesQueryName(tx *gorm.DB, inventoryID string) *gorm.DB {
	query := systemAdvisoriesQuery(tx).Where("sp.inventory_id = ?", inventoryID)
	return query
}

func SystemAdvisoriesQueryByID(tx *gorm.DB, systemID int) *gorm.DB {
	query := systemAdvisoriesQuery(tx).Where("sp.id = ?", systemID)
	return query
}

func systemAdvisoriesQuery(tx *gorm.DB) *gorm.DB {
	query := tx.Table("system_advisories sa").Select("sa.*").
		Joins("join advisory_metadata am ON am.id=sa.advisory_id").
		Joins("join system_platform sp ON sa.system_id=sp.id")
	return query
}
