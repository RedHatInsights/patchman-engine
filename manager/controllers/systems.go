package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
	"strings"
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
	DefaultSort: "-last_upload",
}

type SystemDBLookup struct {
	ID string `query:"sp.inventory_id"`
	SystemItemAttributes
}

type SystemItemAttributes struct {
	DisplayName    string     `json:"display_name" csv:"display_name" query:"sp.display_name"`
	LastEvaluation *time.Time `json:"last_evaluation" csv:"last_evaluation" query:"sp.last_evaluation"`
	LastUpload     *time.Time `json:"last_upload" csv:"last_upload" query:"sp.last_upload"`
	RhsaCount      int        `json:"rhsa_count" csv:"rhsa_count" query:"sp.advisory_sec_count_cache"`
	RhbaCount      int        `json:"rhba_count" csv:"rhba_count" query:"sp.advisory_bug_count_cache"`
	RheaCount      int        `json:"rhea_count" csv:"rhea_count" query:"sp.advisory_enh_count_cache"`
	Stale          bool       `json:"stale" csv:"stale" query:"sp.stale"`

	PackagesInstalled int `json:"packages_installed" csv:"packages_installed" query:"sp.packages_installed"`
	PackagesUpdatable int `json:"packages_updatable" csv:"packages_updatable" query:"sp.packages_updatable"`
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
// @Param    tags                    query   []string  false "Tag filter"
// @Param    tags                    query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]   query string   false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string false "Filter systems by their SAP SIDs"
// @Success 200 {object} SystemsResponse
// @Router /api/patch/v1/systems [get]
func SystemsListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	query := querySystems(account)
	query = ApplySearch(c, query, "sp.display_name")
	query, _ = ApplyTagsFilter(c, query, "sp.inventory_id")
	query, meta, links, err := ListCommon(query, c, "/api/patch/v1/systems", SystemOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

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

// nolint: gocritic
// @Summary Export systems for my account
// @Description  Export systems for my account
// @ID exportSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[display_name]    query   string  false "Filter"
// @Param    filter[last_evaluation] query   string  false "Filter"
// @Param    filter[last_upload]     query   string  false "Filter"
// @Param    filter[rhsa_count]      query   string  false "Filter"
// @Param    filter[rhba_count]      query   string  false "Filter"
// @Param    filter[rhea_count]      query   string  false "Filter"
// @Param    filter[enabled]         query   string  false "Filter"
// @Param    filter[stale]           query   string  false "Filter"
// @Success 200 {array} SystemInlineItem
// @Router /api/patch/v1/export/systems [get]
func SystemsExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	query := querySystems(account)

	var systems []SystemDBLookup

	query = query.Order("id")
	query, err := ExportListCommon(query, c, SystemOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	data := make([]SystemInlineItem, len(systems))

	for i, v := range systems {
		data[i] = SystemInlineItem(v)
	}

	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") {
		c.JSON(http.StatusOK, data)
	} else if strings.Contains(accept, "text/csv") {
		Csv(c, 200, data)
	} else {
		LogWarnAndResp(c, http.StatusUnsupportedMediaType,
			fmt.Sprintf("Invalid content type '%s', use 'application/json' or 'text/csv'", accept))
	}
}

func querySystems(account int) *gorm.DB {
	return database.Db.Table("system_platform sp").Select(SystemsSelect).
		Where("sp.rh_account_id = ?", account)
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
