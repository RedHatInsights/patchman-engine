package controllers

import (
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
// @Success 200 {array} AdvisoryInlineItemV3
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/advisories [get]
func AdvisoriesExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	filters, err := ParseTagsFilters(c)
	if err != nil {
		return
	}
	db := middlewares.DBFromContext(c)
	var query *gorm.DB
	if disableCachedCounts || HasTags(c) {
		var err error
		query = buildQueryAdvisoriesTagged(db, filters, account)
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

	apiver := c.GetInt(middlewares.KeyApiver)
	if apiver < 3 {
		dataV2 := makeAdvisoryInlineItemV2(advisories)
		OutputExportData(c, dataV2)
		return
	}

	data := make([]AdvisoryInlineItemV3, len(advisories))
	for i, v := range advisories {
		v.SystemAdvisoryItemAttributes = systemAdvisoryItemAttributeParse(v.SystemAdvisoryItemAttributes)
		data[i] = AdvisoryInlineItemV3{
			AdvisoryID:               v.AdvisoryID,
			AdvisoryItemAttributesV3: v.AdvisoryItemAttributesV3,
		}
	}

	OutputExportData(c, data)
}

func makeAdvisoryInlineItemV2(advisories []AdvisoriesDBLookupV3) []AdvisoryInlineItemV2 {
	dataV2 := make([]AdvisoryInlineItemV2, len(advisories))
	for i, v := range advisories {
		v.SystemAdvisoryItemAttributes = systemAdvisoryItemAttributeParse(v.SystemAdvisoryItemAttributes)
		dataV2[i] = AdvisoryInlineItemV2{
			AdvisoryID: v.AdvisoryID,
			AdvisoryItemAttributesV2: AdvisoryItemAttributesV2{
				SystemAdvisoryItemAttributes: v.SystemAdvisoryItemAttributes,
				AdvisoryItemAttributesV2Only: AdvisoryItemAttributesV2Only{
					// this is not typo, v2 applicable_systems are instalable systems in v3
					ApplicableSystems: v.InstallableSystems,
				},
			},
		}
	}
	return dataV2
}
