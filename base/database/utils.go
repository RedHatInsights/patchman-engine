package database

import "github.com/jinzhu/gorm"

func SystemAdvisoriesQueryName(tx *gorm.DB, accountID int, inventoryID string) *gorm.DB {
	query := systemAdvisoriesQuery(tx, accountID).Where("sp.inventory_id = ?::uuid", inventoryID)
	return query
}

func SystemAdvisoriesQueryByID(tx *gorm.DB, accountID, systemID int) *gorm.DB {
	query := systemAdvisoriesQuery(tx, accountID).Where("sp.id = ?", systemID)
	return query
}

func systemAdvisoriesQuery(tx *gorm.DB, accountID int) *gorm.DB {
	query := tx.Table("system_advisories sa").Select("sa.*").
		Joins("join advisory_metadata am ON am.id=sa.advisory_id").
		Joins("join system_platform sp ON sa.rh_account_id = sp.rh_account_id AND sa.system_id = sp.id").
		Where("sa.rh_account_id = ? AND sp.rh_account_id = ?", accountID, accountID)
	return query
}
