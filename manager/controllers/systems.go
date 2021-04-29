package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
)

var SystemsFields = database.MustGetQueryAttrs(&SystemDBLookup{})
var SystemsSelect = database.MustGetSelect(&SystemDBLookup{})
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
}

type SystemDBLookup struct {
	ID string `query:"sp.inventory_id" gorm:"column:id"`
	SystemItemAttributes
}

// nolint: lll
type SystemItemAttributes struct {
	DisplayName    string     `json:"display_name" csv:"display_name" query:"sp.display_name" gorm:"column:display_name"`
	LastEvaluation *time.Time `json:"last_evaluation" csv:"last_evaluation" query:"sp.last_evaluation" gorm:"column:last_evaluation"`
	LastUpload     *time.Time `json:"last_upload" csv:"last_upload" query:"sp.last_upload" gorm:"column:last_upload"`
	RhsaCount      int        `json:"rhsa_count" csv:"rhsa_count" query:"sp.advisory_sec_count_cache" gorm:"column:advisory_sec_count_cache"`
	RhbaCount      int        `json:"rhba_count" csv:"rhba_count" query:"sp.advisory_bug_count_cache" gorm:"column:advisory_bug_count_cache"`
	RheaCount      int        `json:"rhea_count" csv:"rhea_count" query:"sp.advisory_enh_count_cache" gorm:"column:advisory_enh_count_cache"`
	Stale          bool       `json:"stale" csv:"stale" query:"sp.stale" gorm:"column:stale"`
	ThirdParty     bool       `json:"third_party" csv:"third_party" query:"sp.third_party" gorm:"column:third_party"`

	PackagesInstalled int `json:"packages_installed" csv:"packages_installed" query:"sp.packages_installed" gorm:"column:packages_installed"`
	PackagesUpdatable int `json:"packages_updatable" csv:"packages_updatable" query:"sp.packages_updatable" gorm:"column:packages_updatable"`

	OSName  string `json:"os_name" csv:"os_name" query:"ih.system_profile->'operating_system'->>'name'" gorm:"column:osname"`
	OSMajor string `json:"os_major" csv:"os_major" query:"ih.system_profile->'operating_system'->>'major'" gorm:"column:osmajor"`
	OSMinor string `json:"os_minor" csv:"os_minor" query:"ih.system_profile->'operating_system'->>'minor'" gorm:"column:osminor"`
	Rhsm    string `json:"rhsm" csv:"rhsm" query:"ih.system_profile->'rhsm'->>'version'" gorm:"column:rhsm"`

	StaleTimestamp        *time.Time `json:"stale_timestamp" csv:"stale_timestamp" query:"ih.stale_timestamp" gorm:"column:stale_timestamp"`
	StaleWarningTimestamp *time.Time `json:"stale_warning_timestamp" csv:"stale_warning_timestamp" query:"ih.stale_warning_timestamp" gorm:"column:stale_warning_timestamp"`
	CulledTimestamp       *time.Time `json:"culled_timestamp" csv:"culled_timestamp" query:"ih.culled_timestamp" gorm:"column:culled_timestamp"`
	Created               *time.Time `json:"created" csv:"created" query:"ih.created" gorm:"column:created"`
}

type SystemItem struct {
	Attributes SystemItemAttributes `json:"attributes"`
	ID         string               `json:"id"`
	Type       string               `json:"type"`
}

type SystemInlineItem struct {
	ID string `json:"id" csv:"id"`
	SystemItemAttributes
}

type SystemsResponse struct {
	Data  []SystemItem `json:"data"`
	Links Links        `json:"links"`
	Meta  ListMeta     `json:"meta"`
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
// @Param    sort    query   string  false   "Sort field" Enums(id,display_name,last_evaluation,last_upload,rhsa_count,rhba_count,rhea_count,stale, packages_installed, packages_updatable)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[display_name]    query   string  false "Filter"
// @Param    filter[last_evaluation] query   string  false "Filter"
// @Param    filter[last_upload]     query   string  false "Filter"
// @Param    filter[rhsa_count]      query   string  false "Filter"
// @Param    filter[rhba_count]      query   string  false "Filter"
// @Param    filter[rhea_count]      query   string  false "Filter"
// @Param    filter[stale]           query   string  false "Filter"
// @Param    filter[packages_installed] query string false "Filter"
// @Param    filter[packages_updatable] query string false "Filter"
// @Param    filter[stale_timestamp] query string false "Filter"
// @Param    filter[stale_warning_timestamp] query string false "Filter"
// @Param    filter[culled_timestamp] query string false "Filter"
// @Param    filter[created] query string false "Filter"
// @Param    tags                    query   []string  false "Tag filter"
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

	data := buildData(systems)
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
		Select(SystemsSelect)
}

func buildData(systems []SystemDBLookup) []SystemItem {
	data := make([]SystemItem, len(systems))
	for i, system := range systems {
		data[i] = SystemItem{
			Attributes: system.SystemItemAttributes,
			ID:         system.ID,
			Type:       "system",
		}
	}
	return data
}
