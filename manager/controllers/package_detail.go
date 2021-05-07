package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

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
	Description string `json:"description" query:"descr.value"`
	Summary     string `json:"summary" query:"sum.value"`
	Name        string `json:"name" query:"pn.name"`
	EVRA        string `json:"version" query:"p.evra"`
	AdvID       string `json:"advisory_id" query:"am.name" gorm:"column:advisory_id"`
}

var PackageSelect = database.MustGetSelect(&PackageDetailAttributes{})

func packageLatestHandler(c *gin.Context, packageName string) {
	if !packageNameIsValid(packageName) {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "invalid package name"})
		return
	}

	query := database.PackageByName(database.Db, packageName)
	var pkg PackageDetailAttributes
	// Perform 'soft-filtering' by ordering on boolean column first
	err := query.Select(PackageSelect).
		Order("(sum.value IS NOT NULL) DESC NULLS LAST, am.public_date DESC").Limit(1).Find(&pkg).Error
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

	query := database.PackageByName(database.Db, nevra.Name)
	var pkg PackageDetailAttributes
	err := query.Select(PackageSelect).Where("p.evra = ?", nevra.EVRAString()).Take(&pkg).Error
	if err != nil {
		LogAndRespNotFound(c, err, "package not found")
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
