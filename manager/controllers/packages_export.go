package controllers

import (
	"app/base/utils"
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
)

// nolint: lll
// @Summary Show me all installed packages across my systems
// @Description Show me all installed packages across my systems
// @ID exportPackages
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    sort           query      string  false   "Sort field" Enums(id,name,systems_installed,systems_installable,systems_applicable)
// @Param    search         query      string  false   "Find matching text"
// @Param    filter[name]    query     string  false "Filter"
// @Param    filter[systems_installed]   query string  false "Filter"
// @Param    filter[systems_installable] query string  false "Filter"
// @Param    filter[systems_applicable]  query string  false "Filter"
// @Param    filter[summary]           query   string  false "Filter"
// @Success 200 {array} PackageItem
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/packages [get]
func PackagesExportHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)
	filters, err := ParseAllFilters(c, PackagesOpts)
	if err != nil {
		return
	}

	db := middlewares.DBFromContext(c)
	useCache := shouldUseCache(db, account, filters, groups)
	if !useCache {
		db.Exec("SET work_mem TO '?'", utils.Cfg.DBWorkMem)
		defer db.Exec("RESET work_mem")
	}
	query := packagesQuery(db, filters, account, groups, useCache)
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

	OutputExportData(c, items)
}
