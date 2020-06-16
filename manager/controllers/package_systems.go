package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"net/http"
)

type PackageSystems []struct {
	InventoryID string `json:"id"`
	Version     string `json:"version"`
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
	// Weird ->> 0 bit taken from
	// https://stackoverflow.com/questions/27215216/postgres-how-to-convert-a-json-string-to-text
	// It's required to get the textual value of JSONB query
	var systems PackageSystems
	err := database.Db.
		Table("system_platform").
		Select("system_platform.inventory_id, package_data -> ? -> 'version' ->> 0 as version", packageName).
		Joins("inner join rh_account ra on system_platform.rh_account_id = ra.id").
		Where("ra.name = ?", account).
		// TODO: this seems to not be accelerated by the gin index, investigate
		Where("jsonb_exists(system_platform.package_data, ?)", packageName).
		Scan(&systems).Error

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	c.JSON(200, systems)
}
