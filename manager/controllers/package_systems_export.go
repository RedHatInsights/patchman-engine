package controllers

import (
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary Show me all my systems which have a package installed
// @Description  Show me all my systems which have a package installed
// @ID exportPackageSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    package_name    path    string    true  "Package name"
// @Param    filter[group_name] 									query []string 	false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]						query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Param    tags            query   []string  false "Tag filter"
// @Success 200 {array} PackageSystemItemV3
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/packages/{package_name}/systems [get]
func PackageSystemsExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	apiver := c.GetInt(middlewares.KeyApiver)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return
	}

	db := middlewares.DBFromContext(c)
	var packageIDs []int
	if err := packagesByNameQuery(db, packageName).Pluck("p.id", &packageIDs).Error; err != nil {
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
	query, _ = ApplyInventoryFilter(inventoryFilters, query, "sp.inventory_id")
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

	outputItems, _ := packageSystemDBLookups2PackageSystemItemsV3(systems)
	if apiver < 3 {
		itemsV2 := packageSystemItemV3toV2(outputItems)
		OutputExportData(c, itemsV2)
		return
	}

	OutputExportData(c, outputItems)
}
