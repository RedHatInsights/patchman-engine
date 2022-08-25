package controllers

import (
	"app/base/database"
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var PackagesFields = database.MustGetQueryAttrs(&PackageItem{})
var PackagesSelect = database.MustGetSelect(&PackageItem{})
var PackagesOpts = ListOpts{
	Fields: PackagesFields,
	// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
	DefaultFilters: map[string]FilterData{},
	DefaultSort:    "name",
	StableSort:     "pn.id",
	SearchFields:   []string{"pn.name", "pn.summary"},
	TotalFunc:      CountRows,
}

//nolint:lll
type PackageItem struct {
	Name             string `json:"name" csv:"name" query:"pn.name" gorm:"column:name"`
	SystemsInstalled int    `json:"systems_installed" csv:"systems_installed" query:"res.systems_installed" gorm:"column:systems_installed"`
	SystemsUpdatable int    `json:"systems_updatable" csv:"systems_updatable" query:"res.systems_updatable" gorm:"column:systems_updatable"`
	Summary          string `json:"summary" csv:"summary" query:"pn.summary" gorm:"column:summary"`
}

type PackagesResponse struct {
	Data  []PackageItem `json:"data"`
	Links Links         `json:"links"`
	Meta  ListMeta      `json:"meta"`
}

// nolint: lll
// Used as a for subquery performing the actual calculation which is joined with latest summaries
type queryItem struct {
	NameID           int `query:"spkg.name_id" gorm:"column:name_id"`
	SystemsInstalled int `json:"systems_installed" query:"count(spkg.system_id)" gorm:"column:systems_installed"`
	SystemsUpdatable int `json:"systems_updatable" query:"count(spkg.system_id) filter (where spkg.latest_evra IS NOT NULL)" gorm:"column:systems_updatable"`
}

var queryItemSelect = database.MustGetSelect(&queryItem{})

func packagesQuery(filters map[string]FilterData, acc int) *gorm.DB {
	systemsWithPkgsInstalledQ := database.Systems(database.Db, acc).
		Select("id").
		Where("sp.stale = false AND sp.packages_installed > 0")

	// We need to apply tag filtering on subquery
	systemsWithPkgsInstalledQ, _ = ApplyTagsFilter(filters, systemsWithPkgsInstalledQ, "sp.inventory_id")
	subQ := database.SystemPackagesShort(database.Db, acc).
		Select(queryItemSelect).
		Where("spkg.system_id IN (?)", systemsWithPkgsInstalledQ).
		Group("spkg.name_id")

	return database.Db.
		Select(PackagesSelect).
		Table("package_name pn").
		Joins("JOIN (?) res ON res.name_id = pn.id", subQ)
}

// @Summary Show me all installed packages across my systems
// @Description Show me all installed packages across my systems
// @ID listPackages
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query      int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query      int     false   "Offset for paging"
// @Param    sort           query      string  false   "Sort field" Enums(id,name,systems_installed,systems_updatable)
// @Param    search         query      string  false   "Find matching text"
// @Param    filter[name]    query     string  false "Filter"
// @Param    filter[systems_installed] query   string  false "Filter"
// @Param    filter[systems_updatable] query   string  false "Filter"
// @Param    filter[summary]           query   string  false "Filter"
// @Param    tags                      query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]					query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} PackagesResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /packages/ [get]
func PackagesListHandler(c *gin.Context) {
	var filters map[string]FilterData
	account := c.GetInt(middlewares.KeyAccount)

	filters, err := ParseTagsFilters(c)
	if err != nil {
		return
	}
	query := packagesQuery(filters, account)
	if err != nil {
		return
	} // Error handled in method itself
	query, meta, links, err := ListCommon(query, c, filters, PackagesOpts)
	if err != nil {
		return
	} // Error handled in method itself

	var packages = make([]PackageItem, 0)
	err = query.Scan(&packages).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	c.JSON(200, PackagesResponse{
		Data:  packages,
		Links: *links,
		Meta:  *meta,
	})
}
