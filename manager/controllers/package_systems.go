package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var PackageSystemFields = database.MustGetQueryAttrs(&PackageSystemDBLookup{})
var PackageSystemsSelect = database.MustGetSelect(&PackageSystemDBLookup{})
var PackageSystemsOpts = ListOpts{
	Fields: PackageSystemFields,
	// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
	DefaultFilters: map[string]FilterData{},
	DefaultSort:    "id",
	SearchFields:   []string{"sp.display_name"},
	TotalFunc:      CountRows,
}

//nolint:lll
type PackageSystemItem struct {
	ID               string         `json:"id" csv:"id" query:"sp.inventory_id" gorm:"column:id"`
	DisplayName      string         `json:"display_name" csv:"display_name" query:"sp.display_name" gorm:"column:display_name"`
	InstalledEVRA    string         `json:"installed_evra" csv:"installed_evra" query:"p.evra" gorm:"column:installed_evra"`
	AvailableEVRA    string         `json:"available_evra" csv:"available_evra" query:"spkg.latest_evra" gorm:"column:available_evra"`
	Updatable        bool           `json:"updatable" csv:"updatable" query:"spkg.latest_evra IS NOT NULL" gorm:"column:updatable"`
	Tags             SystemTagsList `json:"tags" csv:"tags" query:"null" gorm:"-"`
	BaselineName     string         `json:"baseline_name" csv:"baseline_name" query:"bl.name" gorm:"column:baseline_name"`
	BaselineUpToDate *bool          `json:"baseline_uptodate" csv:"baseline_uptodate" query:"sp.baseline_uptodate" gorm:"column:baseline_uptodate"`
}

type PackageSystemDBLookup struct {
	// Just helper field to get tags from db in plain string, then parsed to "Tags" attr., excluded from output data.
	TagsStr string `json:"-" csv:"-" query:"ih.tags" gorm:"column:tags_str"`

	PackageSystemItem
}

type PackageSystemsResponse struct {
	Data  []PackageSystemItem `json:"data"`
	Links Links               `json:"links"`
	Meta  ListMeta            `json:"meta"`
}

func packagesByNameQuery(pkgName string) *gorm.DB {
	return database.Db.Table("package p").
		Joins("INNER JOIN package_name pn ON p.name_id = pn.id").
		Where("pn.name = ?", pkgName)
}

func packageSystemsQuery(acc int, packageName string, packageIDs []int) *gorm.DB {
	query := database.SystemPackages(database.Db, acc).
		Select(PackageSystemsSelect).
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Joins("LEFT JOIN baseline bl ON sp.baseline_id = bl.id AND sp.rh_account_id = bl.rh_account_id").
		Where("sp.stale = false").
		Where("pn.name = ?", packageName).
		Where("spkg.package_id in (?)", packageIDs)

	return query
}

//nolint: dupl
// @Summary Show me all my systems which have a package installed
// @Description  Show me all my systems which have a package installed
// @ID packageSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    package_name    path    string    true  "Package name"
// @Param    tags            query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]					query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} PackageSystemsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /packages/{package_name}/systems [get]
func PackageSystemsListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	var filters map[string]FilterData

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return
	}

	var packageIDs []int
	if err := packagesByNameQuery(packageName).Pluck("p.id", &packageIDs).Error; err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	if len(packageIDs) == 0 {
		LogAndRespNotFound(c, errors.New("not found"), "package not found")
		return
	}

	query := packageSystemsQuery(account, packageName, packageIDs)
	filters, err := ParseTagsFilters(c)
	if err != nil {
		return
	} // Error handled in method itself
	query, _ = ApplyTagsFilter(filters, query, "sp.inventory_id")
	query, meta, links, err := ListCommon(query, c, filters, PackageSystemsOpts)
	if err != nil {
		return
	} // Error handled in method itself

	var systems []PackageSystemDBLookup
	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	outputItems := packageSystemDBLookups2PackageSystemItems(systems)
	c.JSON(200, PackageSystemsResponse{
		Data:  outputItems,
		Links: *links,
		Meta:  *meta,
	})
}

func packageSystemDBLookups2PackageSystemItems(systems []PackageSystemDBLookup) []PackageSystemItem {
	data := make([]PackageSystemItem, len(systems))
	var err error
	for i, system := range systems {
		system.PackageSystemItem.Tags, err = parseSystemTags(system.TagsStr)
		if err != nil {
			utils.Log("err", err.Error(), "inventory_id", system.ID).Debug("system tags parsing failed")
		}
		data[i] = system.PackageSystemItem
	}
	return data
}
