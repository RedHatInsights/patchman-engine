package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

type PackageSystemItem struct {
	InventoryID string `json:"id"`
	EVRA        string `json:"evra"`
}

type PackageSystemsResponse struct {
	Data  []PackageSystemItem `json:"data"`
	Links Links               `json:"links"`
	Meta  ListMeta            `json:"meta"`
}

func packageSystemsQuery(acc int, pkgName string) *gorm.DB {
	return database.Db.
		Table("system_platform sp").
		// nolint: lll
		Joins("inner join system_package spkg on spkg.system_id = sp.id and sp.rh_account_id = ?", acc).
		Joins("inner join package p on p.id = spkg.package_id").
		Joins("inner join package_name pn on pn.id = p.name_id").
		Where("spkg.rh_account_id = ?", acc).
		Where("pn.name = ?", pkgName).
		Select("sp.inventory_id, p.evra as evra")
}

// @Summary Show me all my systems which have a package installed
// @Description  Show me all my systems which have a package installed
// @ID packageSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    package_name    path    string    true  "Package name"
// @Param    tags            query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]   query  string  false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string  false "Filter systems by their SAP SIDs"
// @Success 200 {object} PackageSystemsResponse
// @Router /api/patch/v1/packages/{package_name}/systems [get]
func PackageSystemsListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return
	}
	query := packageSystemsQuery(account, packageName)
	query, meta, links, err := ListCommon(query, c, fmt.Sprintf("/packages/%s/systems", packageName), SystemOpts)
	query = ApplySearch(c, query, "sp.display_name")
	query, _ = ApplyTagsFilter(c, query, "sp.inventory_id")

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}
	var systems []PackageSystemItem
	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	c.JSON(200, PackageSystemsResponse{
		Data:  systems,
		Links: *links,
		Meta:  *meta,
	})
}
