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
	Version     string `json:"version"`
}

type PackageSystemsResponse struct {
	Data  []PackageSystemItem `json:"data"`
	Links Links               `json:"links"`
	Meta  ListMeta            `json:"meta"`
}

func packageSystemsQuery(acc string, pkgName string) *gorm.DB {
	// Weird ->> 0 bit taken from
	// https://stackoverflow.com/questions/27215216/postgres-how-to-convert-a-json-string-to-text
	// It's required to get the textual value of JSONB query
	return database.Db.
		Table("system_platform").
		Joins("inner join rh_account ra on system_platform.rh_account_id = ra.id").
		Joins("inner join system_package spkg on spkg.system_id = system_platform.id").
		Joins("inner join package p on p.id = spkg.package_id").
		Where("ra.name = ?", acc).
		Where("p.name = ?", pkgName).
		Select("system_platform.inventory_id, spkg.version_installed as version")
}

// @Summary Show me all my systems which have a package installed
// @Description  Show me all my systems which have a package installed
// @ID packageSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    package_name    path    string   true "Package name"
// @Success 200 {object} PackageSystems
// @Router /api/patch/v1/packages/{package_name}/systems [get]
func PackageSystemsListHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return
	}
	query := packageSystemsQuery(account, packageName)
	query, meta, links, err := ListCommon(query, c, fmt.Sprintf("/packages/%s/systems", packageName), SystemOpts)

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}
	var systems []PackageSystemItem
	err = query.Scan(&systems).Error
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
