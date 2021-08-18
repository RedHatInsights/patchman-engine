package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var PackageVersionFields = database.MustGetQueryAttrs(&PackageVersionItem{})
var PackageVersionSelect = database.MustGetSelect(&PackageVersionItem{})
var PackageVersionsOpts = ListOpts{
	Fields: PackageVersionFields,
	// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
	DefaultFilters: map[string]FilterData{},
	DefaultSort:    "evra",
	SearchFields:   []string{"p.evra"},
	TotalFunc:      CountRows,
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

// @Summary Show me all package versions installed on some system
// @Description Show me all package versions installed on some system
// @ID packageVersions
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    package_name    path    string    true  "Package name"
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
	// we don't support tags and filters for this endpoint
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
