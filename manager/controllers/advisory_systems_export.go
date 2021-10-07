package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// @Summary Export systems for my account
// @Description  Export systems for my account
// @ID exportAdvisorySystems
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    advisory_id    path    string  true    "Advisory ID"
// @Param    search         query   string  false   "Find matching text"
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
// @Param    filter[system_profile][sap_system]   query string   false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string false "Filter systems by their SAP SIDs"
// @Param    filter[osname] query string false "Filter"
// @Param    filter[osminor] query string false "Filter"
// @Param    filter[osmajor] query string false "Filter"
// @Param    filter[os]              query   string    false "Filter OS version"
// @Param    tags                    query   []string  false "Tag filter"
// @Success 200 {array} SystemInlineItem
// @Router /api/patch/v1/export/advisories/{advisory_id}/systems [get]
func AdvisorySystemsExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	advisoryName := c.Param("advisory_id")
	if advisoryName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "advisory_id param not found"})
		return
	}

	var exists int64
	err := database.Db.Model(&models.AdvisoryMetadata{}).
		Where("name = ? ", advisoryName).Count(&exists).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
	}
	if exists == 0 {
		LogAndRespNotFound(c, errors.New("advisory not found"), "Advisory not found")
		return
	}

	query := buildAdvisorySystemsQuery(account, advisoryName)
	query, _, err = ApplyTagsFilter(c, query, "sp.inventory_id")
	if err != nil {
		return
	} // Error handled in method itself

	var systems []SystemDBLookup

	query = query.Order("sp.id")
	query, err = ExportListCommon(query, c, AdvisorySystemOpts)
	if err != nil {
		return
	} // Error handled in method itself

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
	if strings.Contains(accept, "application/json") { // nolint: gocritic
		c.JSON(http.StatusOK, data)
	} else if strings.Contains(accept, "text/csv") {
		Csv(c, 200, data)
	} else {
		LogWarnAndResp(c, http.StatusUnsupportedMediaType,
			fmt.Sprintf("Invalid content type '%s', use 'application/json' or 'text/csv'", accept))
	}
}
