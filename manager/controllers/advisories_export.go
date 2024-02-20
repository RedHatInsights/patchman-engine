package controllers

import (
	"app/base/utils"
	"app/manager/middlewares"

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
// @Success 200 {array} AdvisoriesDBLookupV3
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/advisories [get]
func AdvisoriesExportHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)
	filters, err := ParseAllFilters(c, AdvisoriesOpts)
	if err != nil {
		return
	}
	db := middlewares.DBFromContext(c)
	var query *gorm.DB

	if disableCachedCounts || HasInventoryFilter(filters) || len(groups) != 0 {
		var err error
		query = buildQueryAdvisoriesTagged(db, filters, account, groups)
		if err != nil {
			return
		} // Error handled in method itself
	} else {
		query = buildQueryAdvisories(db, account)
	}

	var advisories []AdvisoriesDBLookupV3

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

	// update release_version field
	for i := range advisories {
		advisories[i].AdvisoryItemAttributesCommon =
			fillAdvisoryItemAttributeReleaseVersion(advisories[i].AdvisoryItemAttributesCommon)
	}

	OutputExportData(c, advisories)
}
