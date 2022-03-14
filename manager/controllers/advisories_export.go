package controllers

import (
	"app/manager/middlewares"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// @Summary Export applicable advisories for all my systems
// @Description  Export applicable advisories for all my systems
// @ID exportAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]                 query   string  false "Filter"
// @Param    filter[description]        query   string  false "Filter"
// @Param    filter[public_date]        query   string  false "Filter"
// @Param    filter[synopsis]           query   string  false "Filter"
// @Param    filter[advisory_type]      query   string  false "Filter"
// @Param    filter[advisory_type_name] query   string  false "Filter"
// @Param    filter[severity]           query   string  false "Filter"
// @Param    filter[applicable_systems] query   string  false "Filter"
// @Success 200 {array} AdvisoryInlineItem
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/patch/v1/export/advisories [get]
func AdvisoriesExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	filters, err := ParseTagsFilters(c)
	if err != nil {
		return
	}
	var query *gorm.DB
	if disableCachedCounts || HasTags(c) {
		var err error
		query = buildQueryAdvisoriesTagged(filters, account)
		if err != nil {
			return
		} // Error handled in method itself
	} else {
		query = buildQueryAdvisories(account)
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
		LogAndRespError(c, err, "db error")
		return
	}

	data := make([]AdvisoryInlineItem, len(advisories))

	for i, v := range advisories {
		v.SystemAdvisoryItemAttributes = systemAdvisoryItemAttributeParse(v.SystemAdvisoryItemAttributes)
		data[i] = AdvisoryInlineItem(v)
	}
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") { // nolint: gocritic
		c.JSON(http.StatusOK, data)
	} else if strings.Contains(accept, "text/csv") {
		Csv(c, http.StatusOK, data)
	} else {
		LogWarnAndResp(c, http.StatusUnsupportedMediaType,
			fmt.Sprintf("Invalid content type '%s', use 'application/json' or 'text/csv'", accept))
	}
}
