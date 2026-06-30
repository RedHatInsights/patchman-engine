package controllers

import (
	"app/base/utils"
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
)

// nolint:lll
// @Summary Export applicable advisories for all my systems
// @Description  Export applicable advisories for all my systems. Export endpoints are not paginated.
// @ID exportAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]                 query   string  false "Filter"
// @Param    filter[description]        query   string  false "Filter"
// @Param    filter[public_date]        query   string  false "Filter"
// @Param    filter[synopsis]           query   string  false "Filter"
// @Param    filter[advisory_type_name] query   string  false "Filter" Enums(unknown,unspecified,other,enhancement,bugfix,security)
// @Param    filter[severity]           query   int     false "Filter" minimum(1) maximum(4)
// @Param    filter[severity_name]      query   string  false "Filter" Enums(Low,Medium,High,Critical)
// @Param    filter[applicable_systems] query   int     false "Filter"
// @Success 200 {array} AdvisoriesDBLookup
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/advisories [get]
func AdvisoriesExportHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	workspaceIDs := c.GetStringSlice(utils.KeyInventoryWorkspaces)
	filters, err := ParseAllFilters(c, AdvisoriesOpts)
	if err != nil {
		return
	}
	db := middlewares.DBFromContext(c)

	query, err := resolveAdvisoriesQuery(db, account, workspaceIDs, filters)
	if err != nil {
		utils.LogAndRespError(c, err, "db error")
		return
	}

	var advisories []AdvisoriesDBLookup

	query = query.Order("id")
	query, err = ExportListCommon(query, c, AdvisoriesOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	err = query.Find(&advisories).Error
	if err != nil {
		utils.LogAndRespError(c, err, "db error")
		return
	}

	OutputExportData(c, advisories)
}
