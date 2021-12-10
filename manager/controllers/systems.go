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
var SystemsSelect = database.MustGetSelect(&SystemDBLookup{})
var SystemsSumFields = database.MustGetSelect(&SystemSums{})
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
	SearchFields: []string{"sp.display_name"},
	TotalFunc:    systemSubtotals,
}

type SystemDBLookup struct {
	ID string `json:"id" csv:"id" query:"sp.inventory_id" gorm:"column:id"`

	// Just helper field to get tags from db in plain string, then parsed to "Tags" attr., excluded from output data.
	TagsStr string `json:"-" csv:"-" query:"ih.tags" gorm:"column:tags_str"`

	SystemItemAttributes
}

type SystemInlineItem SystemDBLookup

// nolint: lll
type SystemItemAttributes struct {
	DisplayName    string     `json:"display_name" csv:"display_name" query:"sp.display_name" gorm:"column:display_name"`
	LastEvaluation *time.Time `json:"last_evaluation" csv:"last_evaluation" query:"sp.last_evaluation" gorm:"column:last_evaluation"`
	LastUpload     *time.Time `json:"last_upload" csv:"last_upload" query:"sp.last_upload" gorm:"column:last_upload"`
	RhsaCount      int        `json:"rhsa_count" csv:"rhsa_count" query:"sp.advisory_sec_count_cache" gorm:"column:rhsa_count"`
	RhbaCount      int        `json:"rhba_count" csv:"rhba_count" query:"sp.advisory_bug_count_cache" gorm:"column:rhba_count"`
	RheaCount      int        `json:"rhea_count" csv:"rhea_count" query:"sp.advisory_enh_count_cache" gorm:"column:rhea_count"`
	OtherCount     int        `json:"other_count" csv:"other_count" query:"(sp.advisory_count_cache - sp.advisory_sec_count_cache - sp.advisory_bug_count_cache - sp.advisory_enh_count_cache)" gorm:"column:other_count"`
	Stale          bool       `json:"stale" csv:"stale" query:"sp.stale" gorm:"column:stale"`
	ThirdParty     bool       `json:"third_party" csv:"third_party" query:"sp.third_party" gorm:"column:third_party"`
	InsightsID     string     `json:"insights_id" csv:"insights_id" query:"ih.insights_id" gorm:"column:insights_id"`

	PackagesInstalled int `json:"packages_installed" csv:"packages_installed" query:"sp.packages_installed" gorm:"column:packages_installed"`
	PackagesUpdatable int `json:"packages_updatable" csv:"packages_updatable" query:"sp.packages_updatable" gorm:"column:packages_updatable"`

	OSName  string `json:"os_name" csv:"os_name" query:"ih.system_profile->'operating_system'->>'name'" gorm:"column:osname"`
	OSMajor string `json:"os_major" csv:"os_major" query:"ih.system_profile->'operating_system'->>'major'" gorm:"column:osmajor"`
	OSMinor string `json:"os_minor" csv:"os_minor" query:"ih.system_profile->'operating_system'->>'minor'" gorm:"column:osminor"`
	OS      string `json:"os" csv:"os" query:"concat(COALESCE(ih.system_profile->'operating_system'->>'name' || ' ',''), COALESCE(ih.system_profile->'operating_system'->>'major', ''), '.' || (ih.system_profile->'operating_system'->>'minor'))" order_query:"ih.system_profile->'operating_system'->>'name',cast(substring(ih.system_profile->'operating_system'->>'major','^\\d+') as int),cast(substring(ih.system_profile->'operating_system'->>'minor','^\\d+') as int)" gorm:"column:os"`
	Rhsm    string `json:"rhsm" csv:"rhsm" query:"ih.system_profile->'rhsm'->>'version'" gorm:"column:rhsm"`

	StaleTimestamp        *time.Time `json:"stale_timestamp" csv:"stale_timestamp" query:"ih.stale_timestamp" gorm:"column:stale_timestamp"`
	StaleWarningTimestamp *time.Time `json:"stale_warning_timestamp" csv:"stale_warning_timestamp" query:"ih.stale_warning_timestamp" gorm:"column:stale_warning_timestamp"`
	CulledTimestamp       *time.Time `json:"culled_timestamp" csv:"culled_timestamp" query:"ih.culled_timestamp" gorm:"column:culled_timestamp"`
	Created               *time.Time `json:"created" csv:"created" query:"ih.created" gorm:"column:created"`

	Tags SystemTagsList `json:"tags" csv:"tags" gorm:"-"`

	BaselineName     string `json:"-" csv:"-" query:"bl.name" gorm:"column:baseline_name"`
	BaselineUpToDate *bool  `json:"-" csv:"-" query:"sp.baseline_uptodate" gorm:"column:baseline_uptodate"`
}

type SystemTagsList []SystemTag

func (v SystemTagsList) String() string {
	b, err := json.Marshal(v)
	if err != nil {
		utils.Log("err", err.Error()).Error("Unable to convert tags struct to json")
	}
	replacedQuotes := strings.ReplaceAll(string(b), `"`, `'`) // use the same way as "vulnerability app"
	return replacedQuotes
}

// nolint: lll
type SystemSums struct {
	Total     int64 `query:"count(*)" gorm:"column:total"`
	Patched   int64 `query:"count(*) filter (where sp.stale = false and sp.packages_updatable = 0)" gorm:"column:patched"`
	Unpatched int64 `query:"count(*) filter (where sp.stale = false and sp.packages_updatable > 0)" gorm:"column:unpatched"`
	Stale     int64 `query:"count(*) filter (where sp.stale = true)" gorm:"column:stale"`
}

type SystemTag struct {
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}

type SystemItem struct {
	Attributes SystemItemAttributes `json:"attributes"`
	ID         string               `json:"id"`
	Type       string               `json:"type"`
}

type SystemsResponse struct {
	Data  []SystemItem `json:"data"`
	Links Links        `json:"links"`
	Meta  ListMeta     `json:"meta"`
}

func systemSubtotals(tx *gorm.DB) (total int, subTotals map[string]int, err error) {
	var sums SystemSums
	err = tx.Select(SystemsSumFields).Scan(&sums).Error
	if err == nil {
		total = int(sums.Total)
		subTotals = map[string]int{
			"patched":   int(sums.Patched),
			"unpatched": int(sums.Unpatched),
			"stale":     int(sums.Stale),
		}
	}
	return total, subTotals, err
}

// nolint: lll
// @Summary Show me all my systems
// @Description Show me all my systems
// @ID listSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit   query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset  query   int     false   "Offset for paging"
// @Param    sort    query   string  false   "Sort field" Enums(id,display_name,last_evaluation,last_upload,rhsa_count,rhba_count,rhea_count,other_count,stale, packages_installed, packages_updatable)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[insights_id]     query   string  false "Filter"
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[display_name]    query   string  false "Filter"
// @Param    filter[last_evaluation] query   string  false "Filter"
// @Param    filter[last_upload]     query   string  false "Filter"
// @Param    filter[rhsa_count]      query   string  false "Filter"
// @Param    filter[rhba_count]      query   string  false "Filter"
// @Param    filter[rhea_count]      query   string  false "Filter"
// @Param    filter[other_count]     query   string  false "Filter"
// @Param    filter[stale]           query   string  false "Filter"
// @Param    filter[packages_installed] query string false "Filter"
// @Param    filter[packages_updatable] query string false "Filter"
// @Param    filter[stale_timestamp] query string false "Filter"
// @Param    filter[stale_warning_timestamp] query string false "Filter"
// @Param    filter[culled_timestamp] query   string    false "Filter"
// @Param    filter[created]          query   string    false "Filter"
// @Param    filter[osname]           query   string false "Filter"
// @Param    filter[osminor]          query   string false "Filter"
// @Param    filter[osmajor]          query   string false "Filter"
// @Param    filter[os]               query   string    false "Filter OS version"
// @Param    tags                     query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]   query string   false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string false "Filter systems by their SAP SIDs"
// @Success 200 {object} SystemsResponse
// @Router /api/patch/v1/systems [get]
func SystemsListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	query := querySystems(account)
	query, _, err := ApplyTagsFilter(c, query, "sp.inventory_id")
	if err != nil {
		return
	} // Error handled method itself
	query, meta, links, err := ListCommon(query, c, "/api/patch/v1/systems", SystemOpts)
	if err != nil {
		return
	} // Error handled method itself

	var systems []SystemDBLookup
	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	data := systemDBLookups2SystemItems(systems)
	resp := SystemsResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func querySystems(account int) *gorm.DB {
	return database.Systems(database.Db, account).
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Joins("LEFT JOIN baseline bl ON sp.baseline_id = bl.id AND sp.rh_account_id = bl.rh_account_id").
		Select(SystemsSelect)
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
