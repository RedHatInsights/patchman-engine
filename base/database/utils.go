package database

import "github.com/jinzhu/gorm"

func Systems(tx *gorm.DB, accountID int) *gorm.DB {
	return tx.Table("system_platform sp").Where("sp.rh_account_id = ?", accountID)
}

func SystemAdvisories(tx *gorm.DB, accountID int) *gorm.DB {
	return Systems(tx, accountID).
		Joins("JOIN system_advisories sa on sa.system_id = sp.id AND sa.rh_account_id = ?", accountID)
}

func SystemPackages(tx *gorm.DB, accountID int) *gorm.DB {
	return Systems(tx, accountID).
		Joins("JOIN system_package spkg on spkg.system_id = sp.id AND spkg.rh_account_id = ?", accountID).
		Joins("JOIN package p on p.id = spkg.package_id").
		Joins("JOIN package_name pn on pn.id = p.name_id")
}

func Packages(tx *gorm.DB) *gorm.DB {
	return tx.Table("package p").
		Joins("JOIN package_name pn on p.name_id = pn.id").
		Joins("JOIN (SELECT id, value FROM strings) descr ON p.description_hash = descr.id").
		Joins("JOIN (SELECT id, value from strings) sum ON p.summary_hash = sum.id").
		Joins("JOIN (SELECT id, name, public_date from advisory_metadata) am ON p.advisory_id = am.id")
}

func PackageByName(tx *gorm.DB, pkgName string) *gorm.DB {
	return Packages(tx).Where("pn.name = ?", pkgName)
}

func SystemAdvisoriesByInventoryID(tx *gorm.DB, accountID int, inventoryID string) *gorm.DB {
	return SystemAdvisories(tx, accountID).Where("sp.inventory_id = ?::uuid", inventoryID)
}

func SystemAdvisoriesBySystemID(tx *gorm.DB, accountID, systemID int) *gorm.DB {
	query := systemAdvisoriesQuery(tx, accountID).Where("sp.id = ?", systemID)
	return query
}

func systemAdvisoriesQuery(tx *gorm.DB, accountID int) *gorm.DB {
	query := tx.Table("system_advisories sa").Select("sa.*").
		Joins("join system_platform sp ON sa.rh_account_id = sp.rh_account_id AND sa.system_id = sp.id").
		Where("sa.rh_account_id = ? AND sp.rh_account_id = ?", accountID, accountID)
	return query
}
