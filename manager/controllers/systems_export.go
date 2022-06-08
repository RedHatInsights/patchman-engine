package controllers

import (
	"app/manager/middlewares"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// @Summary Export systems for my account
// @Description  Export systems for my account
// @ID exportSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[display_name]    query   string  false "Filter"
// @Param    filter[last_evaluation] query   string  false "Filter"
// @Param    filter[last_upload]     query   string  false "Filter"
// @Param    filter[rhsa_count]      query   string  false "Filter"
// @Param    filter[rhba_count]      query   string  false "Filter"
// @Param    filter[rhea_count]      query   string  false "Filter"
// @Param    filter[other_count]     query   string  false "Filter"
// @Param    filter[stale]           query   string  false "Filter"
// @Param    filter[packages_installed] query string false "Filter"
// @Param    filter[packages_updatable] query string false "Filter"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]					query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Param    filter[osname]          query   string false "Filter"
// @Param    filter[osminor]         query   string false "Filter"
// @Param    filter[osmajor]         query   string false "Filter"
// @Param    filter[baseline_name]   query   string false "Filter"
// @Param    filter[os]              query   string    false "Filter OS version"
// @Param    tags                    query   []string  false "Tag filter"
// @Success 200 {array} SystemInlineItem
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/systems [get]
func SystemsExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	query := querySystems(account)
	filters, err := ParseTagsFilters(c)
	if err != nil {
		return
	} // Error handled in method itself
	query, _ = ApplyTagsFilter(filters, query, "sp.inventory_id")

	var systems []SystemDBLookup

	query = query.Order("sp.id")
	query, err = ExportListCommon(query, c, SystemOpts)
	if err != nil {
		return
	} // Error handled in method itself

	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	parseAndFillTags(&systems)
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
