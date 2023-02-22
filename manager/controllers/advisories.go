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
var AdvisoriesOpts = ListOpts{
	Fields:         AdvisoriesFields,
	DefaultFilters: nil,
	DefaultSort:    "-public_date",
	StableSort:     "id",
	SearchFields:   []string{"am.name", "am.cve_list", "synopsis"},
}

type AdvisoryID struct {
	ID string `query:"am.name" gorm:"column:id"`
}

// nolint: lll
type AdvisoriesDBLookup struct {
	ID string `query:"am.name" gorm:"column:id"`
	// a helper to get total number of systems
	MetaTotalHelper
	TotalOther       int `json:"-" csv:"-" query:"count(*) filter (where am.advisory_type_id not in (1,2,3)) over ()" gorm:"column:total_other"`
	TotalEnhancement int `json:"-" csv:"-" query:"count(*) filter (where am.advisory_type_id = 1) over ()" gorm:"column:total_enhancement"`
	TotalBugfix      int `json:"-" csv:"-" query:"count(*) filter (where am.advisory_type_id = 2) over ()" gorm:"column:total_bugfix"`
	TotalSecurty     int `json:"-" csv:"-" query:"count(*) filter (where am.advisory_type_id = 3) over ()" gorm:"column:total_security"`

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

type AdvisoriesResponse struct {
	Data  []AdvisoryItem `json:"data"`
	Links Links          `json:"links"`
	Meta  ListMeta       `json:"meta"`
}

func advisoriesCommon(db *gorm.DB, c *gin.Context) (*gorm.DB, *ListMeta, []string, error) {
	account := c.GetInt(middlewares.KeyAccount)
	var query *gorm.DB
	filters, err := ParseTagsFilters(c)
	if err != nil {
		return nil, nil, nil, err
	}
	if disableCachedCounts || HasTags(c) {
		var err error
		query = buildQueryAdvisoriesTagged(db, filters, account)
		if err != nil {
			return nil, nil, nil, err
		} // Error handled in method itself
	} else {
		query = buildQueryAdvisories(db, account)
	}

	query, meta, params, err := ListCommon(query, c, filters, AdvisoriesOpts)
	// Error handling and setting of result code & content is done in ListCommon
	return query, meta, params, err
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
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]					query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} AdvisoriesResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /advisories [get]
func AdvisoriesListHandler(c *gin.Context) {
	db := middlewares.DBFromContext(c)
	query, meta, params, err := advisoriesCommon(db, c)
	if err != nil {
		return
	} // Error handled in method itself

	var advisories []AdvisoriesDBLookup
	err = query.Find(&advisories).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
	}
	data, total, subtotals := buildAdvisoriesData(advisories)
	meta, links, err := UpdateMetaLinks(c, meta, total, subtotals, params...)
	if err != nil {
		return // Error handled in method itself
	}
	var resp = AdvisoriesResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

// nolint:lll
// @Summary Show me all applicable advisories for all my systems
// @Description Show me all applicable advisories for all my systems
// @ID listAdvisoriesIds
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
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]					query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} IDsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /ids/advisories [get]
func AdvisoriesListIDsHandler(c *gin.Context) {
	db := middlewares.DBFromContext(c)
	query, _, _, err := advisoriesCommon(db, c)
	if err != nil {
		return
	} // Error handled in method itself
	var aids []AdvisoryID
	err = query.Find(&aids).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
	}

	ids := advisoriesIDs(aids)
	var resp = IDsResponse{IDs: ids}
	c.JSON(http.StatusOK, &resp)
}

func buildQueryAdvisories(db *gorm.DB, account int) *gorm.DB {
	query := db.Table("advisory_metadata am").
		Select(AdvisoriesSelect).
		Joins("JOIN advisory_account_data aad ON am.id = aad.advisory_id and aad.systems_affected > 0").
		Joins("JOIN advisory_type at ON am.advisory_type_id = at.id").
		Where("aad.rh_account_id = ?", account)
	return query
}

func buildAdvisoryAccountDataQuery(db *gorm.DB, account int) *gorm.DB {
	query := database.SystemAdvisories(db, account).
		Select("sa.advisory_id, sp.rh_account_id as rh_account_id, 0 as status_id, count(sp.id) as systems_affected").
		Where("sp.stale = false").
		Group("sp.rh_account_id, sa.advisory_id")

	return query
}

func buildQueryAdvisoriesTagged(db *gorm.DB, filters map[string]FilterData, account int) *gorm.DB {
	subq := buildAdvisoryAccountDataQuery(db, account)
	subq, _ = ApplyTagsFilter(filters, subq, "sp.inventory_id")

	query := db.Table("advisory_metadata am").
		Select(AdvisoriesSelect).
		Joins("JOIN advisory_type at ON am.advisory_type_id = at.id").
		Joins("JOIN (?) aad ON am.id = aad.advisory_id and aad.systems_affected > 0", subq)

	return query
}

func buildAdvisoriesData(advisories []AdvisoriesDBLookup) ([]AdvisoryItem, int, map[string]int) {
	var total int
	subtotals := map[string]int{
		"other":       0,
		"enhancement": 0,
		"bugfix":      0,
		"security":    0,
	}
	if len(advisories) > 0 {
		total = advisories[0].Total
		subtotals["other"] = advisories[0].TotalOther
		subtotals["enhancement"] = advisories[0].TotalEnhancement
		subtotals["bugfix"] = advisories[0].TotalBugfix
		subtotals["security"] = advisories[0].TotalSecurty
	}
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
	return data, total, subtotals
}
