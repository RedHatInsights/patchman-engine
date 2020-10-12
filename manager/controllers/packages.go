package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

var PackagesFields = database.MustGetQueryAttrs(&PackageItem{})
var PackagesSelect = database.MustGetSelect(&PackageItem{})
var PackagesOpts = ListOpts{
	Fields: PackagesFields,
	// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
	DefaultFilters: map[string]FilterData{},
	DefaultSort:    "name",
}

// nolint: lll
type PackageItem struct {
	Name             string `json:"name" query:"pn.name"`
	SystemsInstalled int    `json:"systems_installed" query:"count(sp.id)"`
	SystemsUpdatable int    `json:"systems_updatable" query:"count(sp.id) filter (where spkg.update_data is not null)"`
	Summary          string `json:"summary" query:"(select s.value from package p inner join strings s on p.summary_hash = s.id inner join advisory_metadata am on p.advisory_id = am.id where p.name_id = pn.id order by am.public_date limit 1)"`
}

type PackagesResponse struct {
	Data  []PackageItem `json:"data"`
	Links Links         `json:"links"`
	Meta  ListMeta      `json:"meta"`
}

func packagesQuery(acc int) *gorm.DB {
	return database.Db.
		Select(PackagesSelect).
		Table("system_platform sp").
		Joins("inner join rh_account ra on sp.rh_account_id = ra.id").
		Joins("inner join system_package spkg on spkg.system_id = sp.id").
		Joins("inner join package p on p.id = spkg.package_id").
		Joins("inner join package_name pn on pn.id = p.name_id").
		Where("spkg.rh_account_id = ?", acc).
		Group("ra.id, pn.id, pn.name")
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
// @Param    tags                      query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]   query string   false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string false "Filter systems by their SAP SIDs"
// @Success 200 {object} PackagesResponse
// @Router /api/patch/v1/packages/ [get]
func PackagesListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	query := packagesQuery(account)
	query, meta, links, err := ListCommon(query, c, "/packages", PackagesOpts)
	query, _ = ApplyTagsFilter(c, query, "sp.inventory_id")

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}
	var systems []PackageItem
	err = query.Scan(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	c.JSON(200, PackagesResponse{
		Data:  systems,
		Links: *links,
		Meta:  *meta,
	})
}
