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
// @Param    filter[applicable_systems] query  string  false "Filter"
// @Param    tags                    query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system] query  string  false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string  false "Filter systems by their SAP SIDs"
// @Success 200 {object} AdvisoriesResponse
// @Router /api/patch/v1/advisories [get]
func AdvisoriesListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	var query *gorm.DB

	if HasTags(c) {
		var err error
		query, err = buildQueryAdvisoriesTagged(c, account)
		if err != nil {
			return
		} // Error handled in method itself
	} else {
		query = buildQueryAdvisories(account)
	}

	query = ApplySearch(c, query, "am.name", "am.cve_list", "synopsis", "description")
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

func buildQueryAdvisories(account int) *gorm.DB {
	query := database.Db.Table("advisory_metadata am").
		Select(AdvisoriesSelect).
		Joins("JOIN advisory_account_data aad ON am.id = aad.advisory_id and aad.systems_affected > 0").
		Where("aad.rh_account_id = ?", account)
	return query
}

// nolint: lll
func buildAdvisoryAccountDataQuery(account int) *gorm.DB {
	query := database.Db.Table("system_advisories sa").
		Select("sa.advisory_id, sp.rh_account_id as rh_account_id, 0 as status_id, count(sp.id) as systems_affected, 0 as systems_status_divergent").
		Joins("join system_platform sp on sp.id = sa.system_id and sp.stale = false").
		Where("sa.when_patched is null").
		Where("sp.stale = false").
		Where("sp.rh_account_id = ? ", account).
		Group("sp.rh_account_id, sa.advisory_id")

	return query
}

func buildQueryAdvisoriesTagged(c *gin.Context, account int) (*gorm.DB, error) {
	subq := buildAdvisoryAccountDataQuery(account)
	subq, _, err := ApplyTagsFilter(c, subq, "sp.inventory_id")
	if err != nil {
		return nil, err
	}

	query := database.Db.Table("advisory_metadata am").
		Select(AdvisoriesSelect).
		Joins("JOIN (?) aad ON am.id = aad.advisory_id and aad.systems_affected > 0", subq.SubQuery())

	return query, nil
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
