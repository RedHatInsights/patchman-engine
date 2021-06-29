package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

var PackageVersionFields = database.MustGetQueryAttrs(&PackageVersionItem{})
var PackageVersionSelect = database.MustGetSelect(&PackageVersionItem{})
var PackageVersionsOpts = ListOpts{
	Fields: PackageVersionFields,
	// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
	DefaultFilters: map[string]FilterData{},
	DefaultSort:    "evra",
	SearchFields:   []string{"p.evra"},
}

type PackageVersionItem struct {
	Evra string `json:"evra" csv:"evra" query:"evra" gorm:"column:evra"`
}

type PackageVersionsResponse struct {
	Data  []PackageVersionItem `json:"data"`
	Links Links                `json:"links"`
	Meta  ListMeta             `json:"meta"`
}

func packagesNameID(pkgName string) *gorm.DB {
	return database.Db.Table("package_name pn").
		Where("pn.name = ?", pkgName)
}

func packageVersionsQuery(acc int, packageNameIDs []int) *gorm.DB {
	query := database.SystemPackages(database.Db, acc).
		Select(PackageVersionSelect).
		Distinct("p.evra").
		Where("sp.stale = false").
		Where("spkg.name_id in (?)", packageNameIDs)
	return query
}

//nolint: dupl
// @Summary Show me all package versions installed on some system
// @Description Show me all package versions installed on some system
// @ID packageVersions
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    package_name    path    string    true  "Package name"
// @Param    tags            query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]   query  string  false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string  false "Filter systems by their SAP SIDs"
// @Success 200 {object} PackageVersionsResponse
// @Router /api/patch/v1/packages/{package_name}/versions [get]
func PackageVersionsListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return
	}

	var packageNameIDs []int
	if err := packagesNameID(packageName).Pluck("pn.id", &packageNameIDs).Error; err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	query := packageVersionsQuery(account, packageNameIDs)
	query, _, err := ApplyTagsFilter(c, query, "sp.inventory_id")
	if err != nil {
		return
	} // Error handled in method itself
	query, meta, links, err := ListCommon(query, c, fmt.Sprintf("/packages/%s/versions", packageName), PackageVersionsOpts)
	if err != nil {
		return
	} // Error handled in method itself

	var systems []PackageVersionItem
	err = query.Find(&systems).Error
	fmt.Println(systems)
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	c.JSON(200, PackageVersionsResponse{
		Data:  systems,
		Links: *links,
		Meta:  *meta,
	})
}
