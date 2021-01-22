package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

func singlePackageQuery(pkgName string) *gorm.DB {
	return database.Db.Table("package pkg").Select("*").
		Joins("JOIN package_name pn ON pkg.name_id = pn.id").
		Joins("JOIN (SELECT id, name as adv_id, public_date FROM advisory_metadata) as adv ON pkg.advisory_id = adv.id").
		Joins("JOIN (SELECT id, value as description FROM strings) as "+
			"string_descr ON pkg.description_hash = string_descr.id").
		Joins("JOIN (SELECT id, value as summary FROM strings) as string_sum ON pkg.summary_hash = string_sum.id").
		Where("pn.name = ?", pkgName)
}

func packageNameIsValid(packageName string) bool {
	var packageNames []models.PackageName
	err := database.Db.Table("package_name").
		Where("name = ?", packageName).
		Find(&packageNames).Error
	if err != nil {
		return false
	}
	return len(packageNames) > 0
}

type PackageDetailResponse struct {
	Data PackageDetailItem `json:"data"`
}

type PackageDetailItem struct {
	Attributes PackageDetailAttributes `json:"attributes"`
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
}

type PackageDetailAttributes struct {
	Description string `json:"description"`
	Summary     string `json:"summary"`
	Name        string `json:"name"`
	EVRA        string `json:"version"`
	AdvID       string `json:"advisory_id"`
}

func packageLatestHandler(c *gin.Context, packageName string) {
	if !packageNameIsValid(packageName) {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "invalid package name"})
		return
	}

	query := singlePackageQuery(packageName)
	var pkg PackageDetailAttributes
	err := query.Order("adv.public_date DESC").Limit(1).Find(&pkg).Error
	if err != nil {
		LogAndRespNotFound(c, err, "package not found")
		return
	}

	nevra := packageName + "-" + pkg.EVRA

	c.JSON(200, PackageDetailResponse{
		Data: PackageDetailItem{
			Attributes: pkg,
			ID:         nevra,
			Type:       "package",
		},
	})
}

func packageEvraHandler(c *gin.Context, nevra *utils.Nevra) {
	if !packageNameIsValid(nevra.Name) {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "invalid package name"})
		return
	}

	query := singlePackageQuery(nevra.Name)
	var pkg PackageDetailAttributes
	rowsReturned := query.Where("pkg.evra = ?", nevra.EVRAString()).Find(&pkg).RowsAffected
	if rowsReturned == 0 {
		LogAndRespNotFound(c, gorm.ErrRecordNotFound, "package not found")
		return
	}

	c.JSON(200, PackageDetailResponse{
		Data: PackageDetailItem{
			Attributes: pkg,
			ID:         nevra.String(),
			Type:       "package",
		},
	})
}

// @Summary Show me metadata of selected package
// @Description Show me metadata of selected package
// @ID LatestPackage
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    package_name    path    string   true "package_name - latest, nevra - exact version"
// @Success 200 {object} PackageDetailResponse
// @Router /api/patch/v1/packages/{package_name} [get]
func PackageDetailHandler(c *gin.Context) {
	parameter := c.Param("package_name")
	if parameter == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_param not found"})
		return
	}

	nevra, err := utils.ParseNevra(parameter)
	if err == nil {
		packageEvraHandler(c, nevra)
	} else {
		packageLatestHandler(c, parameter)
	}
}
