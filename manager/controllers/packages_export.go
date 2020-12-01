package controllers

import (
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// nolint: gocritic
// @Summary Show me all installed packages across my systems
// @Description Show me all installed packages across my systems
// @ID exportPackages
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    sort           query      string  false   "Sort field" Enums(id,name,systems_installed,systems_updatable)
// @Param    search         query      string  false   "Find matching text"
// @Param    filter[name]    query     string  false "Filter"
// @Param    filter[systems_installed] query   string  false "Filter"
// @Param    filter[systems_updatable] query   string  false "Filter"
// @Param    filter[summary]           query   string  false "Filter"
// @Param    tags                      query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]   query string   false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string false "Filter systems by their SAP SIDs"
// @Success 200 {array} PackageItem
// @Router /api/patch/v1/export/packages [get]
func PackagesExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	query, err := packagesQuery(c, account)
	if err != nil {
		return
	}

	query, err = ExportListCommon(query, c, PackagesOpts)
	var data []PackageItem

	if err != nil {
		return
	} // Error handled in method itself

	err = query.Find(&data).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") {
		c.JSON(http.StatusOK, data)
	} else if strings.Contains(accept, "text/csv") {
		Csv(c, 200, data)
	} else {
		LogWarnAndResp(c, http.StatusUnsupportedMediaType,
			fmt.Sprintf("Invalid content type '%s', use 'application/json' or 'text/csv'", accept))
	}
}
