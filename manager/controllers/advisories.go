package controllers

import (
	"app/base/database"
	"app/base/rbac"
	"app/manager/middlewares"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var AdvisoriesFields = database.MustGetQueryAttrs(&AdvisoriesDBLookupV3{})
var AdvisoriesSelectV2 = database.MustGetSelect(&AdvisoriesDBLookupV2{})
var AdvisoriesSelectV3 = database.MustGetSelect(&AdvisoriesDBLookupV3{})
var AdvisoriesOpts = ListOpts{
	Fields: AdvisoriesFields,
	DefaultFilters: map[string]FilterData{
		"applicable_systems": {
			Operator: "gt",
			Values:   []string{"0"},
		},
	},
	DefaultSort:  "-public_date",
	StableSort:   "id",
	SearchFields: []string{"am.name", "am.cve_list", "synopsis"},
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

// nolint:lll
type AdvisoryItemAttributesCommon struct {
	Description      string    `json:"description" csv:"description" query:"am.description" gorm:"column:description"`
	PublicDate       time.Time `json:"public_date" csv:"public_date" query:"am.public_date" gorm:"column:public_date"`
	Synopsis         string    `json:"synopsis" csv:"synopsis" query:"am.synopsis" gorm:"column:synopsis"`
	AdvisoryTypeName string    `json:"advisory_type_name" csv:"advisory_type_name" query:"at.name" order_query:"at.preference" gorm:"column:advisory_type_name"` // Advisory type name, proper ordering ensured (unknown, unspecified, other, enhancement, bugfix, security)
	Severity         *int      `json:"severity,omitempty" csv:"severity" query:"am.severity_id" gorm:"column:severity"`
	CveCount         int       `json:"cve_count" csv:"cve_count" query:"CASE WHEN jsonb_typeof(am.cve_list) = 'array' THEN jsonb_array_length(am.cve_list) ELSE 0 END" gorm:"column:cve_count"`
	RebootRequired   bool      `json:"reboot_required" csv:"reboot_required" query:"am.reboot_required" gorm:"column:reboot_required"`
	ReleaseVersions  RelList   `json:"release_versions" csv:"release_versions" query:"null" gorm:"-"`

	// helper field to get release_version json from db and parse it to ReleaseVersions field
	ReleaseVersionsJSONB []byte `json:"-" csv:"-" query:"am.release_versions" gorm:"column:release_versions_json"`
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
	AdvisoryItemAttributesCommon
	AdvisoryItemAttributesV2Only
}

type AdvisoryItemAttributesV3 struct {
	AdvisoryItemAttributesCommon
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
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)
	var query *gorm.DB
	filters, err := ParseInventoryFilters(c, AdvisoriesOpts)
	if err != nil {
		return nil, nil, nil, err
	}

	var validCache bool
	if !disableCachedCounts {
		err = db.Table("rh_account").
			Select("valid_advisory_cache").
			Where("id = ?", account).
			Scan(&validCache).Error
		if err != nil {
			validCache = false
		}
	}
	if !validCache || HasInventoryFilter(filters) || len(groups[rbac.KeyGrouped]) != 0 {
		var err error
		query = buildQueryAdvisoriesTagged(db, filters, account, groups)
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
// @Param    sort           query   string  false   "Sort field"    Enums(id,advisory_type_name,synopsis,public_date,severity,installable_systems,applicable_systems)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]                  query   string  false "Filter "
// @Param    filter[description]         query   string  false "Filter"
// @Param    filter[public_date]         query   string  false "Filter"
// @Param    filter[synopsis]            query   string  false "Filter"
// @Param    filter[advisory_type_name]  query   string  false "Filter"
// @Param    filter[severity]            query   string  false "Filter"
// @Param    filter[installable_systems] query   string  false "Filter"
// @Param    filter[applicable_systems]  query   string  false "Filter"
// @Param    tags                        query   []string  false "Tag filter"
// @Param    filter[group_name][in]									query string 	false "Filter systems by inventory groups"
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
// @Param    filter[group_name][in]									query string 	false "Filter systems by inventory groups"
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

func buildAdvisoryAccountDataQuery(db *gorm.DB, account int, groups map[string]string) *gorm.DB {
	query := database.SystemAdvisories(db, account, groups).
		Select(`sa.advisory_id, sp.rh_account_id as rh_account_id,
		        count(sp.*) filter (where sa.status_id = 0) as systems_installable,
		        count(sp.*) as systems_applicable`).
		Where("sp.stale = false").
		Group("sp.rh_account_id, sa.advisory_id")

	return query
}

func buildQueryAdvisoriesTagged(db *gorm.DB, filters map[string]FilterData, account int, groups map[string]string,
) *gorm.DB {
	subq := buildAdvisoryAccountDataQuery(db, account, groups)
	subq, _ = ApplyInventoryFilter(filters, subq, "sp.inventory_id")

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
		advisory.AdvisoryItemAttributesCommon = fillAdvisoryItemAttributeReleaseVersion(advisory.AdvisoryItemAttributesCommon)
		data[i] = AdvisoryItemV3{
			Attributes: AdvisoryItemAttributesV3{
				AdvisoryItemAttributesCommon: advisory.AdvisoryItemAttributesCommon,
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
				AdvisoryItemAttributesCommon: items[i].Attributes.AdvisoryItemAttributesCommon,
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
