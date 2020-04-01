package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

var AdvisoriesFields = database.MustGetQueryAttrs(&AdvisoriesDBLookup{})
var AdvisoriesSelect = database.MustGetSelect(&AdvisoriesDBLookup{})

type AdvisoriesDBLookup struct {
	ID string `query:"am.name"`
	AdvisoryItemAttributes
}

type AdvisoryItemAttributes struct {
	SystemAdvisoryItemAttributes
	ApplicableSystems int `json:"applicable_systems" query:"COALESCE(aad.systems_affected, 0)"`
}

type AdvisoryItem struct {
	Attributes AdvisoryItemAttributes `json:"attributes"`
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
}

type AdvisoriesResponse struct {
	Data  []AdvisoryItem `json:"data"`
	Links Links          `json:"links"`
	Meta  ListMeta       `json:"meta"`
}

// nolint:lll
// @Summary Show me all applicable advisories for all my systems
// @Description Show me all applicable advisories for all my systems
// @ID listAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,name,advisory_type,synopsis,public_date,applicable_systems)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[description]     query   string  false "Filter"
// @Param    filter[public_date]     query   string  false "Filter"
// @Param    filter[synopsis]        query   string  false "Filter"
// @Param    filter[advisory_type]   query   string  false "Filter"
// @Param    filter[severity]        query   string  false "Filter"
// @Param    filter[applicable_systems] query   string  false "Filter"
// @Success 200 {object} AdvisoriesResponse
// @Router /api/patch/v1/advisories [get]
func AdvisoriesListHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	query := buildQueryAdvisories(account)

	query = ApplySearch(c, query, "am.name", "synopsis", "description")
	query, meta, links, err := ListCommon(query, c, "/api/patch/v1/advisories", AdvisoriesFields, nil)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var advisories []AdvisoriesDBLookup

	err = query.Find(&advisories).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	data := buildAdvisoriesData(advisories)
	var resp = AdvisoriesResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildQueryAdvisories(account string) *gorm.DB {
	query := database.Db.Table("advisory_metadata am").
		Select(AdvisoriesSelect).
		Joins("JOIN advisory_account_data aad ON am.id = aad.advisory_id and aad.systems_affected > 0").
		Joins("JOIN rh_account ra ON aad.rh_account_id = ra.id").
		Where("ra.name = ?", account)
	return query
}

func buildAdvisoriesData(advisories []AdvisoriesDBLookup) []AdvisoryItem {
	data := make([]AdvisoryItem, len(advisories))
	for i := 0; i < len(advisories); i++ {
		advisory := (advisories)[i]
		data[i] = AdvisoryItem{
			Attributes: AdvisoryItemAttributes{
				SystemAdvisoryItemAttributes: advisory.SystemAdvisoryItemAttributes,
				ApplicableSystems:            advisory.ApplicableSystems,
			},
			ID:   advisory.ID,
			Type: "advisory",
		}
	}
	return data
}
