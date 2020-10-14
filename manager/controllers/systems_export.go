package controllers

import (
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// nolint: gocritic
// @Summary Export systems for my account
// @Description  Export systems for my account
// @ID exportSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
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
// @Success 200 {array} SystemInlineItem
// @Router /api/patch/v1/export/systems [get]
func SystemsExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	query := querySystems(account)
	query = ApplySearch(c, query, "system_platform.display_name")
	query, _ = ApplyTagsFilter(c, query, "system_platform.inventory_id")

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
