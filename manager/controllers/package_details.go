package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

type PackageDetailsItem struct {
	Name string `json:"name" query:"pn.name"`
	EVRA string `json:"evra" query:"pkg.evra"`
	Description string
	Summary string
}

func packageEvraQuery(name, evra string) *gorm.DB {
	return database.Db.Table("package pkg").
		Joins("JOIN package_name pn on pn.id = pkg.name_id").
		Where("pn.name = ?", name).
		Where("pkg.evra = ? ", evra)
}

func PackageEvraDetailHandler(c *gin.Context) {

	//account := c.GetString(middlewares.KeyAccount)

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return
	}

	evra := c.Param("evra")
	if evra == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "evra param not found"})
		return
	}

	tx := packageEvraQuery(packageName, evra)

	var pkg struct {
		models.Package
		Name string
	}
	tx.Find()
}

func PackageLatestDetailHandler(c *gin.Context) {
	//account := c.GetString(middlewares.KeyAccount)

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return
	}

}