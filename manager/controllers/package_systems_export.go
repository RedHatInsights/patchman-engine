package controllers

import (
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// @Summary Show me all my systems which have a package installed
// @Description  Show me all my systems which have a package installed
// @ID exportPackageSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    package_name    path    string    true  "Package name"
// @Param    filter[system_profile][sap_system]   query string   false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string false "Filter systems by their SAP SIDs"
// @Param    tags            query   []string  false "Tag filter"
// @Success 200 {array} PackageSystemItem
// @Router /api/patch/v1/export/packages/{package_name}/systems [get]
func PackageSystemsExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return
	}

	var packageIDs []int
	if err := packagesByNameQuery(packageName).Pluck("p.id", &packageIDs).Error; err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	if len(packageIDs) == 0 {
		LogAndRespNotFound(c, errors.New("not found"), "package not found")
		return
	}

	query := packageSystemsQuery(account, packageName, packageIDs)
	query, _, err := ApplyTagsFilter(c, query, "sp.inventory_id")
	if err != nil {
		return
	} // Error handled in method itself
	query, err = ExportListCommon(query, c, PackageSystemsOpts)
	if err != nil {
		return
	} // Error handled in method itself

	var systems []PackageSystemItem
	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") { // nolint: gocritic
		c.JSON(http.StatusOK, systems)
	} else if strings.Contains(accept, "text/csv") {
		Csv(c, 200, systems)
	} else {
		LogWarnAndResp(c, http.StatusUnsupportedMediaType,
			fmt.Sprintf("Invalid content type '%s', use 'application/json' or 'text/csv'", accept))
	}
}
