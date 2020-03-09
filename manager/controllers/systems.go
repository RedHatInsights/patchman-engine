package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

var SystemsFields = database.MustGetQueryAttrs(&SystemDBLookup{})
var SystemsSelect = database.MustGetSelect(&SystemDBLookup{})

// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
var systemsDefaultFilter = map[string]FilterData{
	"stale": {
		Operator: "eq",
		Values:   []string{"false"},
	},
}

type SystemDBLookup struct {
	ID string `query:"system_platform.inventory_id"`
	SystemItemAttributes
}

type SystemItemAttributes struct {
	LastEvaluation *time.Time `json:"last_evaluation" query:"system_platform.last_evaluation"`
	LastUpload     *time.Time `json:"last_upload" query:"system_platform.last_upload"`
	RhsaCount      int        `json:"rhsa_count" query:"system_platform.advisory_sec_count_cache"`
	RhbaCount      int        `json:"rhba_count" query:"system_platform.advisory_bug_count_cache"`
	RheaCount      int        `json:"rhea_count" query:"system_platform.advisory_enh_count_cache"`
	Enabled        bool       `json:"enabled" query:"(NOT system_platform.opt_out)"`
	Stale          bool       `json:"stale" query:"system_platform.stale"`
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

// nolint: lll
// @Summary Show me all my systems
// @Description Show me all my systems
// @ID listSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit   query   int     false   "Limit for paging"
// @Param    offset  query   int     false   "Offset for paging"
// @Param    sort    query   string  false   "Sort field" Enums(id,last_evaluation,last_updated,rhsa_count,rhba_count,rhea_count,enabled,stale)
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[last_evaluation] query   string  false "Filter"
// @Param    filter[last_updated]    query   string  false "Filter"
// @Param    filter[rhsa_count]      query   string  false "Filter"
// @Param    filter[rhba_count]      query   string  false "Filter"
// @Param    filter[rhea_count]      query   string  false "Filter"
// @Param    filter[enabled]         query   string  false "Filter"
// @Param    filter[stale]           query   string  false "Filter"
// @Success 200 {object} SystemsResponse
// @Router /api/patch/v1/systems [get]
func SystemsListHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	query := database.Db.Table("system_platform").Select(SystemsSelect).
		Joins("inner join rh_account ra on system_platform.rh_account_id = ra.id").
		Where("ra.name = ?", account)

	query, meta, links, err := ListCommon(query, c, "/api/patch/v1/systems", SystemsFields, systemsDefaultFilter)
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
	var resp = SystemsResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
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
