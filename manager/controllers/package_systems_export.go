package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary Show me all my systems which have a package installed
// @Description  Show me all my systems which have a package installed. Export endpoints are not paginated.
// @ID exportPackageSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    package_name    path    string    true  "Package name"
// @Param    filter[group_name] 									query []string 	false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]						query bool  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]						query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Param    tags            query   []string  false "Tag filter"
// @Success 200 {array} PackageSystemItem
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/packages/{package_name}/systems [get]
func PackageSystemsExportHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return
	}

	db := middlewares.DBFromContext(c)
	var packageIDs []int
	if err := database.PackageByName(db, packageName).Pluck("p.id", &packageIDs).Error; err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	if len(packageIDs) == 0 {
		LogAndRespNotFound(c, errors.New("not found"), "package not found")
		return
	}

	query := packageSystemsQuery(db, account, groups, packageName, packageIDs)
	filters, err := ParseAllFilters(c, PackageSystemsOpts)
	if err != nil {
		return
	} // Error handled in method itself
	query, _ = ApplyInventoryFilter(filters, query, "sp.inventory_id")
	query, err = ExportListCommon(query, c, PackageSystemsOpts)
	if err != nil {
		return
	} // Error handled in method itself

	var systems []PackageSystemDBLookup
	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	outputItems, _ := packageSystemDBLookups2PackageSystemItems(systems)
	OutputExportData(c, outputItems)
}
