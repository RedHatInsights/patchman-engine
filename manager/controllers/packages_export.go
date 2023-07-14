package controllers

import (
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
)

// @Summary Show me all installed packages across my systems
// @Description Show me all installed packages across my systems
// @ID exportPackages
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    sort           query      string  false   "Sort field" Enums(id,name,systems_installed,systems_updatable)
// @Param    search         query      string  false   "Find matching text"
// @Param    filter[name]    query     string  false "Filter"
// @Param    filter[systems_installed] query   string  false "Filter"
// @Param    filter[systems_updatable] query   string  false "Filter"
// @Param    filter[summary]           query   string  false "Filter"
// @Success 200 {array} PackageItem
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/packages [get]
func PackagesExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	apiver := c.GetInt(middlewares.KeyApiver)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)
	filters, err := ParseInventoryFilters(c)
	if err != nil {
		return
	}

	db := middlewares.DBFromContext(c)
	query := packagesQuery(db, filters, account, groups)
	if err != nil {
		return
	}

	query, err = ExportListCommon(query, c, PackagesOpts)
	var data []PackageDBLookup

	if err != nil {
		return
	} // Error handled in method itself

	err = query.Find(&data).Error
	items, _ := PackageDBLookup2Item(data)
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}
	if apiver < 3 {
		itemsV2 := packages2PackagesV2(items)
		OutputExportData(c, itemsV2)
		return
	}

	OutputExportData(c, items)
}
