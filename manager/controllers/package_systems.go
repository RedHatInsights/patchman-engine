package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"net/http"
)

type PackageSystems []int

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

	var systemIds models.SystemPlatform
	err := database.Db.
		Joins("inner join rh_account ra on system_platform.rh_account_id = ra.id").
		Where("ra.name = ?", account).
		// TODO: We can't use '?' here, find another way to query for containing this key
		Where("system_platform.package_data ? ?", packageName).
		Pluck("inventory_id", &systemIds).Error

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	c.JSON(200, systemIds)
}
