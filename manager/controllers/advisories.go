package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"net/http"
	"strings"
)

var AdvisoriesFields = database.MustGetQueryAttrs(&AdvisoriesDBLookup{})
var AdvisoriesSelect = database.MustGetSelect(&AdvisoriesDBLookup{})
var AdvisoriesOpts = ListOpts{
	Fields:         AdvisoriesFields,
	DefaultFilters: nil,
	DefaultSort:    "-public_date",
}

type AdvisoriesDBLookup struct {
	ID string `query:"am.name"`
	AdvisoryItemAttributes
}

type AdvisoryItemAttributes struct {
	SystemAdvisoryItemAttributes
	ApplicableSystems int `json:"applicable_systems" query:"COALESCE(aad.systems_affected, 0)" csv:"applicable_systems"`
}

type AdvisoryItem struct {
	Attributes AdvisoryItemAttributes `json:"attributes"`
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
}

type AdvisoryInlineItem struct {
	ID string `json:"id" csv:"id"`
	AdvisoryItemAttributes
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
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
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
	query, meta, links, err := ListCommon(query, c, "/api/patch/v1/advisories", AdvisoriesOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var advisories []AdvisoriesDBLookup

	err = query.Find(&advisories).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
	}

	data := buildAdvisoriesData(advisories)
	var resp = AdvisoriesResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

// nolint: gocritic, lll
// @Summary Export applicable advisories for all my systems
// @Description  Export applicable advisories for all my systems
// @ID exportAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json, text/csv
// @Success 200 {array} AdvisoryInlineItem
// @Router /api/patch/v1/export/advisories [get]
func AdvisoriesExportHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)
	query := buildQueryAdvisories(account)

	var advisories []AdvisoriesDBLookup

	err := query.Find(&advisories).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	data := make([]AdvisoryInlineItem, len(advisories))

	for i, v := range advisories {
		data[i] = AdvisoryInlineItem(v)
	}
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") {
		c.JSON(http.StatusOK, data)
	} else if strings.Contains(accept, "text/csv") {
		Csv(c, 200, data)
	} else {
		LogAndRespStatusError(c, http.StatusUnsupportedMediaType, errors.New("Invalid content type"),
			"This endpoint provides only application/json and text/csv")
		return
	}
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
