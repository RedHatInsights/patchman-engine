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

type SystemsSatelliteManagedID struct {
	ID string `query:"sp.inventory_id" gorm:"column:id"`
	SystemSatelliteManaged
	MetaTotalHelper
}

// nolint: lll
type SystemDBLookupCommon struct {
	SystemIDAttribute
	SystemsMetaTagTotal
	TotalPatched   int `json:"-" csv:"-" query:"count(*) filter (where sp.stale = false and sp.packages_applicable = 0) over ()" gorm:"column:total_patched"`
	TotalUnpatched int `json:"-" csv:"-" query:"count(*) filter (where sp.stale = false and sp.packages_applicable > 0) over ()" gorm:"column:total_unpatched"`
	TotalStale     int `json:"-" csv:"-" query:"count(*) filter (where sp.stale = true) over ()" gorm:"column:total_stale"`
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

	LastEvaluation *time.Time `json:"last_evaluation" csv:"last_evaluation" query:"sp.last_evaluation" gorm:"column:last_evaluation"`
	RhsaCount      int        `json:"rhsa_count" csv:"rhsa_count" query:"sp.installable_advisory_sec_count_cache" gorm:"column:rhsa_count"`
	RhbaCount      int        `json:"rhba_count" csv:"rhba_count" query:"sp.installable_advisory_bug_count_cache" gorm:"column:rhba_count"`
	RheaCount      int        `json:"rhea_count" csv:"rhea_count" query:"sp.installable_advisory_enh_count_cache" gorm:"column:rhea_count"`
	OtherCount     int        `json:"other_count" csv:"other_count" query:"(sp.installable_advisory_count_cache - sp.installable_advisory_sec_count_cache - sp.installable_advisory_bug_count_cache - sp.installable_advisory_enh_count_cache)" gorm:"column:other_count"`

	PackagesInstalled int `json:"packages_installed" csv:"packages_installed" query:"sp.packages_installed" gorm:"column:packages_installed"`

	BaselineNameAttr

	SystemLastUpload
	SystemTimestamps
	SystemStale
	SystemSatelliteManaged
	SystemBuiltPkgcache
}

// nolint: lll
type SystemItemAttributesV2Only struct {
	ThirdParty        bool   `json:"third_party" csv:"third_party" query:"sp.third_party" gorm:"column:third_party"`
	InsightsID        string `json:"insights_id" csv:"insights_id" query:"ih.insights_id" gorm:"column:insights_id"`
	PackagesUpdatable int    `json:"packages_updatable" csv:"packages_updatable" query:"sp.packages_installable" gorm:"column:packages_updatable"`

	OSName  string `json:"os_name" csv:"os_name" query:"ih.system_profile->'operating_system'->>'name'" gorm:"column:osname"`
	OSMajor string `json:"os_major" csv:"os_major" query:"ih.system_profile->'operating_system'->>'major'" gorm:"column:osmajor"`
	OSMinor string `json:"os_minor" csv:"os_minor" query:"ih.system_profile->'operating_system'->>'minor'" gorm:"column:osminor"`
	BaselineUpToDateAttr
}

// nolint: lll
type SystemItemAttributesV3Only struct {
	PackagesInstallable   int `json:"packages_installable" csv:"packages_installable" query:"sp.packages_installable" gorm:"column:packages_installable"`
	PackagesApplicable    int `json:"packages_applicable" csv:"packages_applicable" query:"sp.packages_applicable" gorm:"column:packages_applicable"`
	InstallableRhsaCount  int `json:"installable_rhsa_count" csv:"installable_rhsa_count" query:"sp.installable_advisory_sec_count_cache" gorm:"column:installable_rhsa_count"`
	InstallableRhbaCount  int `json:"installable_rhba_count" csv:"installable_rhba_count" query:"sp.installable_advisory_bug_count_cache" gorm:"column:installable_rhba_count"`
	InstallableRheaCount  int `json:"installable_rhea_count" csv:"installable_rhea_count" query:"sp.installable_advisory_enh_count_cache" gorm:"column:installable_rhea_count"`
	InstallableOtherCount int `json:"installable_other_count" csv:"installable_other_count" query:"(sp.installable_advisory_count_cache - sp.installable_advisory_sec_count_cache - sp.installable_advisory_bug_count_cache - sp.installable_advisory_enh_count_cache)" gorm:"column:installable_other_count"`
	ApplicableRhsaCount   int `json:"applicable_rhsa_count" csv:"applicable_rhsa_count" query:"sp.applicable_advisory_sec_count_cache" gorm:"column:applicable_rhsa_count"`
	ApplicableRhbaCount   int `json:"applicable_rhba_count" csv:"applicable_rhba_count" query:"sp.applicable_advisory_bug_count_cache" gorm:"column:applicable_rhba_count"`
	ApplicableRheaCount   int `json:"applicable_rhea_count" csv:"applicable_rhea_count" query:"sp.applicable_advisory_enh_count_cache" gorm:"column:applicable_rhea_count"`
	ApplicableOtherCount  int `json:"applicable_other_count" csv:"applicable_other_count" query:"(sp.applicable_advisory_count_cache - sp.installable_advisory_sec_count_cache - sp.installable_advisory_bug_count_cache - sp.installable_advisory_enh_count_cache)" gorm:"column:applicable_other_count"`
	BaselineIDAttr
	SystemGroups
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
type SystemGroupsList []SystemGroup
type SystemInventoryItemList[T SystemTagsList | SystemGroupsList] struct {
	SystemInventoryItems T
}

func (v SystemTagsList) String() string {
	return SystemInventoryItemList[SystemTagsList]{v}.String()
}

func (v SystemGroupsList) String() string {
	return SystemInventoryItemList[SystemGroupsList]{v}.String()
}

func (v SystemInventoryItemList[T]) String() string {
	b, err := json.Marshal(v.SystemInventoryItems)
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

type SystemItemV3 struct {
	Attributes SystemItemAttributesV3 `json:"attributes"`
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
}

type SystemsResponseV3 struct {
	Data  []SystemItemV3 `json:"data"`
	Links Links          `json:"links"`
	Meta  ListMeta       `json:"meta"`
}

func systemsCommon(c *gin.Context) (*gorm.DB, *ListMeta, []string, error) {
	var err error
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)
	db := middlewares.DBFromContext(c)
	query := querySystems(db, account, groups)
	filters, err := ParseAllFilters(c, SystemOpts)
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
// @Param    sort       query   string  false   "Sort field" Enums(id,display_name,last_upload,rhsa_count,rhba_count,rhea_count,other_count,stale,packages_installed,baseline_name,groups,satellite_managed,built_pkgcache)
// @Param    search     query   string  false   "Find matching text"
// @Param    filter[id]                     query   string  false   "Filter"
// @Param    filter[display_name]           query   string  false   "Filter"
// @Param    filter[last_evaluation]        query   string  false   "Filter"
// @Param    filter[last_upload]            query   string  false   "Filter"
// @Param    filter[rhsa_count]             query   string  false   "Filter"
// @Param    filter[rhba_count]             query   string  false   "Filter"
// @Param    filter[rhea_count]             query   string  false   "Filter"
// @Param    filter[other_count]            query   string  false   "Filter"
// @Param    filter[installable_rhsa_count] query   string  false   "Filter"
// @Param    filter[installable_rhba_count] query   string  false   "Filter"
// @Param    filter[installable_rhea_count] query   string  false   "Filter"
// @Param    filter[installable_other_count] query   string  false   "Filter"
// @Param    filter[applicable_rhsa_count]  query   string  false   "Filter"
// @Param    filter[applicable_rhba_count]  query   string  false   "Filter"
// @Param    filter[applicable_rhea_count]  query   string  false   "Filter"
// @Param    filter[applicable_other_count] query   string  false   "Filter"
// @Param    filter[stale]                  query   string  false   "Filter"
// @Param    filter[packages_installed]     query   string  false   "Filter"
// @Param    filter[packages_installable]   query   string  false   "Filter"
// @Param    filter[packages_applicable]    query   string  false   "Filter"
// @Param    filter[stale_timestamp]        query   string  false   "Filter"
// @Param    filter[stale_warning_timestamp] query  string  false   "Filter"
// @Param    filter[culled_timestamp]       query   string  false   "Filter"
// @Param    filter[created]                query   string  false   "Filter"
// @Param    filter[baseline_name]          query   string  false   "Filter"
// @Param    filter[satellite_managed] 		query   string  false   "Filter"
// @Param    filter[built_pkgcache]         query   string  false   "Filter"
// @Param    filter[os]                     query   string  false   "Filter OS version"
// @Param    tags                           query   []string false  "Tag filter"
// @Param    filter[group_name] 									query   []string false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]                     query   string  false   "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]	                    query   []string false  "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]                        query   string  false   "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]    query   string  false   "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]                          query   string  false   "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]                 query   string  false   "Filter systems by mssql version"
// @Success 200 {object} SystemsResponseV3
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /systems [get]
func SystemsListHandler(c *gin.Context) {
	query, meta, params, err := systemsCommon(c)
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
// @Param    sort       query   string  false   "Sort field" Enums(id,display_name,last_upload,rhsa_count,rhba_count,rhea_count,other_count,stale,packages_installed,baseline_name,satellite_managed,built_pkgcache)
// @Param    search     query   string  false   "Find matching text"
// @Param    filter[id]                     query   string  false   "Filter"
// @Param    filter[display_name]           query   string  false   "Filter"
// @Param    filter[last_evaluation]        query   string  false   "Filter"
// @Param    filter[last_upload]            query   string  false   "Filter"
// @Param    filter[rhsa_count]             query   string  false   "Filter"
// @Param    filter[rhba_count]             query   string  false   "Filter"
// @Param    filter[rhea_count]             query   string  false   "Filter"
// @Param    filter[other_count]            query   string  false   "Filter"
// @Param    filter[installable_rhsa_count] query   string  false   "Filter"
// @Param    filter[installable_rhba_count] query   string  false   "Filter"
// @Param    filter[installable_rhea_count] query   string  false   "Filter"
// @Param    filter[installable_other_count] query   string  false   "Filter"
// @Param    filter[applicable_rhsa_count]  query   string  false   "Filter"
// @Param    filter[applicable_rhba_count]  query   string  false   "Filter"
// @Param    filter[applicable_rhea_count]  query   string  false   "Filter"
// @Param    filter[applicable_other_count] query   string  false   "Filter"
// @Param    filter[stale]                  query   string  false   "Filter"
// @Param    filter[packages_installed]     query   string  false   "Filter"
// @Param    filter[packages_installable]   query   string  false   "Filter"
// @Param    filter[packages_applicable]    query   string  false   "Filter"
// @Param    filter[stale_timestamp]        query   string  false   "Filter"
// @Param    filter[stale_warning_timestamp] query  string  false   "Filter"
// @Param    filter[culled_timestamp]       query   string  false   "Filter"
// @Param    filter[created]                query   string  false   "Filter"
// @Param    filter[baseline_name]          query   string  false   "Filter"
// @Param    filter[os]                     query   string  false   "Filter OS version"
// @Param    filter[satellite_managed]      query   string  false   "Filter"
// @Param    filter[built_pkgcache]         query   string  false   "Filter"
// @Param    tags                           query   []string false  "Tag filter"
// @Param    filter[group_name] 									query	[]string 	false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]                     query   string  false   "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]	                    query   []string false  "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]                        query   string  false   "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]    query   string  false   "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]                          query   string  false   "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]                 query   string  false   "Filter systems by mssql version"
// @Success 200 {object} IDsSatelliteManagedResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /ids/systems [get]
func SystemsListIDsHandler(c *gin.Context) {
	query, meta, _, err := systemsCommon(c)
	if err != nil {
		return
	} // Error handled in method itself

	var sids []SystemsSatelliteManagedID

	if err = query.Scan(&sids).Error; err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	resp, err := systemsSatelliteIDs(c, sids, meta)
	if err != nil {
		return // Error handled in method itself
	}
	c.JSON(http.StatusOK, &resp)
}

func querySystems(db *gorm.DB, account int, groups map[string]string) *gorm.DB {
	q := database.Systems(db, account, groups).
		Joins("LEFT JOIN baseline bl ON sp.baseline_id = bl.id AND sp.rh_account_id = bl.rh_account_id")
	return q.Select(SystemsSelectV3)
}

func parseSystemItems(jsonStr string, res interface{}) error {
	js := json.RawMessage(jsonStr)
	b, err := json.Marshal(js)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, res); err != nil {
		return err
	}
	return nil
}
