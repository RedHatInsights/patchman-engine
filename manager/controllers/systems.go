package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var SystemsFields = database.MustGetQueryAttrs(&SystemDBLookup{})
var SystemsSelectV2 = database.MustGetSelect(&SystemDBLookupV2{})
var SystemsSelectV3 = database.MustGetSelect(&SystemDBLookupV3{})
var SystemOpts = ListOpts{
	Fields: SystemsFields,
	// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
	DefaultFilters: map[string]FilterData{
		"stale": {
			Operator: "eq",
			Values:   []string{"false"},
		},
	},
	DefaultSort:  "-last_upload",
	StableSort:   "id",
	SearchFields: []string{"sp.display_name"},
}

type SystemsID struct {
	ID string `query:"sp.inventory_id" gorm:"column:id"`
	MetaTotalHelper
}

// nolint: lll
type SystemDBLookupCommon struct {
	SystemIDAttribute
	SystemsMetaTagTotal
	TotalPatched   int `json:"-" csv:"-" query:"count(*) filter (where sp.stale = false and sp.packages_updatable = 0) over ()" gorm:"column:total_patched"`
	TotalUnpatched int `json:"-" csv:"-" query:"count(*) filter (where sp.stale = false and sp.packages_updatable > 0) over ()" gorm:"column:total_unpatched"`
	TotalStale     int `json:"-" csv:"-" query:"count(*) filter (where sp.stale = true) over ()" gorm:"column:total_stale"`
}

type SystemDBLookupV2 struct {
	SystemDBLookupCommon
	SystemItemAttributesV2
}

type SystemDBLookupV3 struct {
	SystemDBLookupCommon
	SystemItemAttributesV3
}

type SystemDBLookup struct {
	SystemDBLookupCommon
	SystemItemAttributesAll
}

// nolint: lll
type SystemItemAttributesCommon struct {
	SystemDisplayName
	OSAttributes
	SystemTags

	RhsaCount  int `json:"rhsa_count" csv:"rhsa_count" query:"sp.installable_advisory_sec_count_cache" gorm:"column:rhsa_count"`
	RhbaCount  int `json:"rhba_count" csv:"rhba_count" query:"sp.installable_advisory_bug_count_cache" gorm:"column:rhba_count"`
	RheaCount  int `json:"rhea_count" csv:"rhea_count" query:"sp.installable_advisory_enh_count_cache" gorm:"column:rhea_count"`
	OtherCount int `json:"other_count" csv:"other_count" query:"(sp.installable_advisory_count_cache - sp.installable_advisory_sec_count_cache - sp.installable_advisory_bug_count_cache - sp.installable_advisory_enh_count_cache)" gorm:"column:other_count"`

	PackagesInstalled int `json:"packages_installed" csv:"packages_installed" query:"sp.packages_installed" gorm:"column:packages_installed"`

	BaselineNameAttr

	SystemLastUpload
	SystemTimestamps
	SystemStale
}

// nolint: lll
type SystemItemAttributesV2Only struct {
	LastEvaluation    *time.Time `json:"last_evaluation" csv:"last_evaluation" query:"sp.last_evaluation" gorm:"column:last_evaluation"`
	ThirdParty        bool       `json:"third_party" csv:"third_party" query:"sp.third_party" gorm:"column:third_party"`
	InsightsID        string     `json:"insights_id" csv:"insights_id" query:"ih.insights_id" gorm:"column:insights_id"`
	PackagesUpdatable int        `json:"packages_updatable" csv:"packages_updatable" query:"sp.packages_updatable" gorm:"column:packages_updatable"`

	OSName  string `json:"os_name" csv:"os_name" query:"ih.system_profile->'operating_system'->>'name'" gorm:"column:osname"`
	OSMajor string `json:"os_major" csv:"os_major" query:"ih.system_profile->'operating_system'->>'major'" gorm:"column:osmajor"`
	OSMinor string `json:"os_minor" csv:"os_minor" query:"ih.system_profile->'operating_system'->>'minor'" gorm:"column:osminor"`
	BaselineUpToDateAttr
}

type SystemItemAttributesV3Only struct {
	BaselineIDAttr
}

type SystemItemAttributesV2 struct {
	SystemItemAttributesCommon
	SystemItemAttributesV2Only
}

type SystemItemAttributesV3 struct {
	SystemItemAttributesCommon
	SystemItemAttributesV3Only
}

type SystemItemAttributesAll struct {
	SystemItemAttributesCommon
	SystemItemAttributesV2Only
	SystemItemAttributesV3Only
}

type SystemTagsList []SystemTag

func (v SystemTagsList) String() string {
	b, err := json.Marshal(v)
	if err != nil {
		utils.LogError("err", err.Error(), "Unable to convert tags struct to json")
	}
	replacedQuotes := strings.ReplaceAll(string(b), `"`, `'`) // use the same way as "vulnerability app"
	return replacedQuotes
}

type SystemItem struct {
	Attributes SystemItemAttributesAll `json:"attributes"`
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
}

type SystemItemV2 struct {
	Attributes SystemItemAttributesV2 `json:"attributes"`
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
}

type SystemItemV3 struct {
	Attributes SystemItemAttributesV3 `json:"attributes"`
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
}

type SystemsResponseV2 struct {
	Data  []SystemItemV2 `json:"data"`
	Links Links          `json:"links"`
	Meta  ListMeta       `json:"meta"`
}

type SystemsResponseV3 struct {
	Data  []SystemItemV3 `json:"data"`
	Links Links          `json:"links"`
	Meta  ListMeta       `json:"meta"`
}

func systemsCommon(c *gin.Context, apiver int) (*gorm.DB, *ListMeta, []string, error) {
	var err error
	account := c.GetInt(middlewares.KeyAccount)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)
	db := middlewares.DBFromContext(c)
	query := querySystems(db, account, apiver, groups)
	filters, err := ParseInventoryFilters(c)
	if err != nil {
		return nil, nil, nil, err
	} // Error handled method itself
	query, _ = ApplyInventoryFilter(filters, query, "sp.inventory_id")
	query, meta, params, err := ListCommon(query, c, filters, SystemOpts)
	// Error handled method itself
	return query, meta, params, err
}

// nolint: lll
// @Summary Show me all my systems
// @Description Show me all my systems
// @ID listSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit      query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset     query   int     false   "Offset for paging"
// @Param    sort       query   string  false   "Sort field" Enums(id,display_name,last_upload,rhsa_count,rhba_count,rhea_count,other_count,stale,packages_installed,baseline_name)
// @Param    search     query   string  false   "Find matching text"
// @Param    filter[id]                     query   string  false   "Filter"
// @Param    filter[display_name]           query   string  false   "Filter"
// @Param    filter[last_upload]            query   string  false   "Filter"
// @Param    filter[rhsa_count]             query   string  false   "Filter"
// @Param    filter[rhba_count]             query   string  false   "Filter"
// @Param    filter[rhea_count]             query   string  false   "Filter"
// @Param    filter[other_count]            query   string  false   "Filter"
// @Param    filter[stale]                  query   string  false   "Filter"
// @Param    filter[packages_installed]     query   string  false   "Filter"
// @Param    filter[stale_timestamp]        query   string  false   "Filter"
// @Param    filter[stale_warning_timestamp] query  string  false   "Filter"
// @Param    filter[culled_timestamp]       query   string  false   "Filter"
// @Param    filter[created]                query   string  false   "Filter"
// @Param    filter[baseline_name]          query   string  false   "Filter"
// @Param    filter[os]                     query   string  false   "Filter OS version"
// @Param    tags                           query   []string false  "Tag filter"
// @Param    filter[system_profile][sap_system]                     query   string  false   "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]                   query   []string false  "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]                        query   string  false   "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]    query   string  false   "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]                          query   string  false   "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]                 query   string  false   "Filter systems by mssql version"
// @Success 200 {object} SystemsResponseV3
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /systems [get]
func SystemsListHandler(c *gin.Context) {
	apiver := c.GetInt(middlewares.KeyApiver)
	query, meta, params, err := systemsCommon(c, apiver)
	if err != nil {
		return
	} // Error handled in method itself

	var systems []SystemDBLookup
	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	data, total, subtotals := systemDBLookups2SystemItems(systems)
	meta, links, err := UpdateMetaLinks(c, meta, total, subtotals, params...)
	if err != nil {
		return // Error handled in method itself
	}
	if apiver < 3 {
		dataV2 := systemItems2SystemItemsV2(data)
		respV2 := SystemsResponseV2{
			Data:  dataV2,
			Links: *links,
			Meta:  *meta,
		}
		c.JSON(http.StatusOK, &respV2)
		return
	}
	dataV3 := systemItems2SystemItemsV3(data)
	resp := SystemsResponseV3{
		Data:  dataV3,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

// nolint: lll
// @Summary Show me all my systems
// @Description Show me all my systems
// @ID listSystemsIds
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit      query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset     query   int     false   "Offset for paging"
// @Param    sort       query   string  false   "Sort field" Enums(id,display_name,last_upload,rhsa_count,rhba_count,rhea_count,other_count,stale,packages_installed,baseline_name)
// @Param    search     query   string  false   "Find matching text"
// @Param    filter[id]                     query   string  false   "Filter"
// @Param    filter[display_name]           query   string  false   "Filter"
// @Param    filter[last_upload]            query   string  false   "Filter"
// @Param    filter[rhsa_count]             query   string  false   "Filter"
// @Param    filter[rhba_count]             query   string  false   "Filter"
// @Param    filter[rhea_count]             query   string  false   "Filter"
// @Param    filter[other_count]            query   string  false   "Filter"
// @Param    filter[stale]                  query   string  false   "Filter"
// @Param    filter[packages_installed]     query   string  false   "Filter"
// @Param    filter[stale_timestamp]        query   string  false   "Filter"
// @Param    filter[stale_warning_timestamp] query  string  false   "Filter"
// @Param    filter[culled_timestamp]       query   string  false   "Filter"
// @Param    filter[created]                query   string  false   "Filter"
// @Param    filter[baseline_name]          query   string  false   "Filter"
// @Param    filter[os]                     query   string  false   "Filter OS version"
// @Param    tags                           query   []string false  "Tag filter"
// @Param    filter[system_profile][sap_system]                     query   string  false   "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]                   query   []string false  "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]                        query   string  false   "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]    query   string  false   "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]                          query   string  false   "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]                 query   string  false   "Filter systems by mssql version"
// @Success 200 {object} IDsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /ids/systems [get]
func SystemsListIDsHandler(c *gin.Context) {
	apiver := c.GetInt(middlewares.KeyApiver)
	query, meta, _, err := systemsCommon(c, apiver)
	if err != nil {
		return
	} // Error handled in method itself

	var sids []SystemsID

	if err = query.Scan(&sids).Error; err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	ids, err := systemsIDs(c, sids, meta)
	if err != nil {
		return // Error handled in method itself
	}
	var resp = IDsResponse{IDs: ids}
	c.JSON(http.StatusOK, &resp)
}

func querySystems(db *gorm.DB, account, apiver int, groups map[string]string) *gorm.DB {
	q := database.Systems(db, account, groups).
		Joins("LEFT JOIN baseline bl ON sp.baseline_id = bl.id AND sp.rh_account_id = bl.rh_account_id")
	if apiver < 3 {
		return q.Select(SystemsSelectV2)
	}
	return q.Select(SystemsSelectV3)
}

func parseSystemTags(jsonStr string) ([]SystemTag, error) {
	js := json.RawMessage(jsonStr)
	b, err := json.Marshal(js)
	if err != nil {
		return nil, err
	}

	var systemTags []SystemTag
	err = json.Unmarshal(b, &systemTags)
	if err != nil {
		return nil, err
	}
	return systemTags, nil
}
