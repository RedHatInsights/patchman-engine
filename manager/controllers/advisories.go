package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var AdvisoriesFields = database.MustGetQueryAttrs(&AdvisoriesDBLookup{})
var AdvisoriesSelect = database.MustGetSelect(&AdvisoriesDBLookup{})
var AdvisoriesSumFields = database.MustGetSelect(&AdvisoriesSums{})
var AdvisoriesOpts = ListOpts{
	Fields:         AdvisoriesFields,
	DefaultFilters: nil,
	DefaultSort:    "-public_date",
	SearchFields:   []string{"am.name", "am.cve_list", "synopsis"},
	TotalFunc:      advisoriesSubtotal,
}

type AdvisoriesDBLookup struct {
	ID string `query:"am.name" gorm:"column:id"`
	AdvisoryItemAttributes
}

// nolint: lll
type AdvisoryItemAttributes struct {
	SystemAdvisoryItemAttributes
	ApplicableSystems int `json:"applicable_systems" query:"COALESCE(aad.systems_affected, 0)" csv:"applicable_systems" gorm:"column:applicable_systems"`
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

type AdvisoriesSums struct {
	Total       int64 `query:"count(*)" gorm:"column:total"`
	Other       int64 `query:"count(*) filter (where am.advisory_type_id not in (1,2,3))" gorm:"column:other"`
	Enhancement int64 `query:"count(*) filter (where am.advisory_type_id = 1)" gorm:"column:enhancement"`
	Bugfix      int64 `query:"count(*) filter (where am.advisory_type_id = 2)" gorm:"column:bugfix"`
	Security    int64 `query:"count(*) filter (where am.advisory_type_id = 3)" gorm:"column:security"`
}

type AdvisoriesResponse struct {
	Data  []AdvisoryItem `json:"data"`
	Links Links          `json:"links"`
	Meta  ListMeta       `json:"meta"`
}

func advisoriesSubtotal(tx *gorm.DB) (total int, subTotals map[string]int, err error) {
	var sums AdvisoriesSums
	err = tx.Select(AdvisoriesSumFields).Scan(&sums).Error
	if err == nil {
		total = int(sums.Total)
		subTotals = map[string]int{
			"other":       int(sums.Other),
			"enhancement": int(sums.Enhancement),
			"bugfix":      int(sums.Bugfix),
			"security":    int(sums.Security),
		}
	}
	return total, subTotals, err
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
// @Param    filter[id]                  query   string  false "Filter "
// @Param    filter[description]         query   string  false "Filter"
// @Param    filter[public_date]         query   string  false "Filter"
// @Param    filter[synopsis]            query   string  false "Filter"
// @Param    filter[advisory_type]       query   string  false "Filter"
// @Param    filter[advisory_type_name]  query   string  false "Filter"
// @Param    filter[severity]            query   string  false "Filter"
// @Param    filter[applicable_systems]  query   string  false "Filter"
// @Param    tags                        query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system] query  string  false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string  false "Filter systems by their SAP SIDs"
// @Success 200 {object} AdvisoriesResponse
// @Router /api/patch/v1/advisories [get]
func AdvisoriesListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	var query *gorm.DB
	filters, err := ParseTagsFilters(c)
	if err != nil {
		return
	}
	if disableCachedCounts || HasTags(c) {
		var err error
		query = buildQueryAdvisoriesTagged(filters, account)
		if err != nil {
			return
		} // Error handled in method itself
	} else {
		query = buildQueryAdvisories(account)
	}

	query, meta, links, err := ListCommon(query, c, filters, "/api/patch/v1/advisories", AdvisoriesOpts)
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

func buildQueryAdvisories(account int) *gorm.DB {
	query := database.Db.Table("advisory_metadata am").
		Select(AdvisoriesSelect).
		Joins("JOIN advisory_account_data aad ON am.id = aad.advisory_id and aad.systems_affected > 0").
		Joins("JOIN advisory_type at ON am.advisory_type_id = at.id").
		Where("aad.rh_account_id = ?", account)
	return query
}

func buildAdvisoryAccountDataQuery(account int) *gorm.DB {
	query := database.SystemAdvisories(database.Db, account).
		Select("sa.advisory_id, sp.rh_account_id as rh_account_id, 0 as status_id, count(sp.id) as systems_affected, " +
			"0 as systems_status_divergent").
		Where("sp.stale = false").
		Group("sp.rh_account_id, sa.advisory_id")

	return query
}

func buildQueryAdvisoriesTagged(filters map[string]FilterData, account int) *gorm.DB {
	subq := buildAdvisoryAccountDataQuery(account)
	subq, _ = ApplyTagsFilter(filters, subq, "sp.inventory_id")

	query := database.Db.Table("advisory_metadata am").
		Select(AdvisoriesSelect).
		Joins("JOIN advisory_type at ON am.advisory_type_id = at.id").
		Joins("JOIN (?) aad ON am.id = aad.advisory_id and aad.systems_affected > 0", subq)

	return query
}

func buildAdvisoriesData(advisories []AdvisoriesDBLookup) []AdvisoryItem {
	data := make([]AdvisoryItem, len(advisories))
	for i := 0; i < len(advisories); i++ {
		advisory := (advisories)[i]
		advisory.SystemAdvisoryItemAttributes = systemAdvisoryItemAttributeParse(advisory.SystemAdvisoryItemAttributes)
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
