package database

import "github.com/jinzhu/gorm"

func SystemAdvisoriesQuery(inventoryId string) *gorm.DB {
	query := Db.Table("advisory_metadata am").Select("am.*").
		Joins("join system_advisories sa ON am.id=sa.advisory_id").
		Joins("join system_platform sp ON sa.system_id=sp.id").
		Where("sp.inventory_id = ?", inventoryId)
	return query
}
