package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var AdvisoriesFields = database.MustGetQueryAttrs(&AdvisoriesDBLookupV3{})
var AdvisoriesSelectV2 = database.MustGetSelect(&AdvisoriesDBLookupV2{})
var AdvisoriesSelectV3 = database.MustGetSelect(&AdvisoriesDBLookupV3{})
var AdvisoriesOpts = ListOpts{
	Fields:         AdvisoriesFields,
	DefaultFilters: nil,
	DefaultSort:    "-public_date",
	StableSort:     "id",
	SearchFields:   []string{"am.name", "am.cve_list", "synopsis"},
}

type AdvisoryID struct {
	ID string `json:"id" csv:"id" query:"am.name" gorm:"column:id"`
}

// nolint: lll
type AdvisoryMetaTotalHelper struct {
	MetaTotalHelper
	TotalOther       int `json:"-" csv:"-" query:"count(*) filter (where am.advisory_type_id not in (1,2,3)) over ()" gorm:"column:total_other"`
	TotalEnhancement int `json:"-" csv:"-" query:"count(*) filter (where am.advisory_type_id = 1) over ()" gorm:"column:total_enhancement"`
	TotalBugfix      int `json:"-" csv:"-" query:"count(*) filter (where am.advisory_type_id = 2) over ()" gorm:"column:total_bugfix"`
	TotalSecurty     int `json:"-" csv:"-" query:"count(*) filter (where am.advisory_type_id = 3) over ()" gorm:"column:total_security"`
}

type AdvisoriesDBLookupCommon struct {
	AdvisoryID
	// a helper to get total number of systems
	AdvisoryMetaTotalHelper
}

type AdvisoriesDBLookupV2 struct {
	AdvisoriesDBLookupCommon
	AdvisoryItemAttributesV2
}

type AdvisoriesDBLookupV3 struct {
	AdvisoriesDBLookupCommon
	AdvisoryItemAttributesV3
}

// nolint: lll
type AdvisoryItemAttributesV2Only struct {
	// this is not typo, v2 applicable_systems are instalable systems in v3
	ApplicableSystems int `json:"applicable_systems" query:"COALESCE(aad.systems_installable, 0)" csv:"applicable_systems" gorm:"column:installable_systems"`
}

// nolint: lll
type AdvisoryItemAttributesV3Only struct {
	InstallableSystems int `json:"installable_systems" query:"COALESCE(aad.systems_installable, 0)" csv:"installable_systems" gorm:"column:installable_systems"`
	ApplicableSystems  int `json:"applicable_systems" query:"COALESCE(aad.systems_applicable, 0)" csv:"applicable_systems" gorm:"column:applicable_systems"`
}

type AdvisoryItemAttributesV2 struct {
	SystemAdvisoryItemAttributes
	AdvisoryItemAttributesV2Only
}

type AdvisoryItemAttributesV3 struct {
	SystemAdvisoryItemAttributes
	AdvisoryItemAttributesV3Only
}

type AdvisoryItemV2 struct {
	Attributes AdvisoryItemAttributesV2 `json:"attributes"`
	AdvisoryID
	Type string `json:"type"`
}

type AdvisoryItemV3 struct {
	Attributes AdvisoryItemAttributesV3 `json:"attributes"`
	AdvisoryID
	Type string `json:"type"`
}

type AdvisoryInlineItemV2 struct {
	AdvisoryID
	AdvisoryItemAttributesV2
}

type AdvisoryInlineItemV3 struct {
	AdvisoryID
	AdvisoryItemAttributesV3
}

type AdvisoriesResponseV2 struct {
	Data  []AdvisoryItemV2 `json:"data"`
	Links Links            `json:"links"`
	Meta  ListMeta         `json:"meta"`
}

type AdvisoriesResponseV3 struct {
	Data  []AdvisoryItemV3 `json:"data"`
	Links Links            `json:"links"`
	Meta  ListMeta         `json:"meta"`
}

func advisoriesCommon(c *gin.Context) (*gorm.DB, *ListMeta, []string, error) {
	db := middlewares.DBFromContext(c)
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
// @Param    filter[installable_systems] query   string  false "Filter"
// @Param    filter[applicable_systems]  query   string  false "Filter"
// @Param    tags                        query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]					query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} AdvisoriesResponseV3
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /advisories [get]
func AdvisoriesListHandler(c *gin.Context) {
	apiver := c.GetInt(middlewares.KeyApiver)
	query, meta, params, err := advisoriesCommon(c)
	if err != nil {
		return
	} // Error handled in method itself

	var advisories []AdvisoriesDBLookupV3
	err = query.Find(&advisories).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
	}
	data, total, subtotals := buildAdvisoriesData(advisories)
	meta, links, err := UpdateMetaLinks(c, meta, total, subtotals, params...)
	if err != nil {
		return // Error handled in method itself
	}
	if apiver < 3 {
		dataV2 := advisoryItemV3toV2(data)
		var resp = AdvisoriesResponseV2{
			Data:  dataV2,
			Links: *links,
			Meta:  *meta,
		}
		c.JSON(http.StatusOK, &resp)
		return
	}
	var resp = AdvisoriesResponseV3{
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
// @Param    filter[installable_systems] query   string  false "Filter"
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
	query, _, _, err := advisoriesCommon(c)
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
		Select(AdvisoriesSelectV3).
		Joins("JOIN advisory_account_data aad ON am.id = aad.advisory_id").
		Joins("JOIN advisory_type at ON am.advisory_type_id = at.id").
		Where("aad.rh_account_id = ?", account)
	return query
}

func buildAdvisoryAccountDataQuery(db *gorm.DB, account int) *gorm.DB {
	query := database.SystemAdvisories(db, account).
		Select(`sa.advisory_id, sp.rh_account_id as rh_account_id,
		        count(sp.*) filter (where sa.status_id = 0) as systems_installable,
		        count(sp.*) filter (where sa.status_id = 1) as systems_applicable`).
		Where("sp.stale = false").
		Group("sp.rh_account_id, sa.advisory_id")

	return query
}

func buildQueryAdvisoriesTagged(db *gorm.DB, filters map[string]FilterData, account int) *gorm.DB {
	subq := buildAdvisoryAccountDataQuery(db, account)
	subq, _ = ApplyTagsFilter(filters, subq, "sp.inventory_id")

	query := db.Table("advisory_metadata am").
		Select(AdvisoriesSelectV3).
		Joins("JOIN advisory_type at ON am.advisory_type_id = at.id").
		Joins("JOIN (?) aad ON am.id = aad.advisory_id", subq)

	return query
}

func buildAdvisoriesData(advisories []AdvisoriesDBLookupV3) ([]AdvisoryItemV3, int, map[string]int) {
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
	data := make([]AdvisoryItemV3, len(advisories))
	for i := 0; i < len(advisories); i++ {
		advisory := (advisories)[i]
		advisory.SystemAdvisoryItemAttributes = systemAdvisoryItemAttributeParse(advisory.SystemAdvisoryItemAttributes)
		data[i] = AdvisoryItemV3{
			Attributes: AdvisoryItemAttributesV3{
				SystemAdvisoryItemAttributes: advisory.SystemAdvisoryItemAttributes,
				AdvisoryItemAttributesV3Only: AdvisoryItemAttributesV3Only{
					InstallableSystems: advisory.InstallableSystems,
					ApplicableSystems:  advisory.ApplicableSystems,
				},
			},
			AdvisoryID: advisory.AdvisoryID,
			Type:       "advisory",
		}
	}
	return data, total, subtotals
}

func advisoryItemV3toV2(items []AdvisoryItemV3) []AdvisoryItemV2 {
	nItems := len(items)
	itemsV2 := make([]AdvisoryItemV2, nItems)
	for i := 0; i < nItems; i++ {
		itemsV2[i] = AdvisoryItemV2{
			Attributes: AdvisoryItemAttributesV2{
				SystemAdvisoryItemAttributes: items[i].Attributes.SystemAdvisoryItemAttributes,
				AdvisoryItemAttributesV2Only: AdvisoryItemAttributesV2Only{
					// this is not typo, v2 applicable_systems are instalable systems in v3
					ApplicableSystems: items[i].Attributes.InstallableSystems,
				},
			},
			AdvisoryID: items[i].AdvisoryID,
			Type:       items[i].Type,
		}
	}
	return itemsV2
}
