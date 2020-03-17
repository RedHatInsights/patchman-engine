package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"net/http"
)

type AdvisorySystemsResponse struct {
	Data  []SystemItem `json:"data"`
	Links Links        `json:"links"`
	Meta  ListMeta     `json:"meta"`
}

// nolint: lll
// @Summary Show me systems on which the given advisory is applicable
// @Description Show me systems on which the given advisory is applicable
// @ID listAdvisorySystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string  true    "Advisory ID"
// @Param    limit          query   int     false   "Limit for paging"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort    query   string  false   "Sort field" Enums(id,last_evaluation,last_updated,rhsa_count,rhba_count,rhea_count,enabled,stale)
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[last_evaluation] query   string  false "Filter"
// @Param    filter[last_updated]    query   string  false "Filter"
// @Param    filter[rhsa_count]      query   string  false "Filter"
// @Param    filter[rhba_count]      query   string  false "Filter"
// @Param    filter[rhea_count]      query   string  false "Filter"
// @Param    filter[enabled]         query   string  false "Filter"
// @Param    filter[stale]           query   string  false "Filter"
// @Success 200 {object} AdvisorySystemsResponse
// @Router /api/patch/v1/advisories/{advisory_id}/systems [get]
func AdvisorySystemsListHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	advisoryName := c.Param("advisory_id")
	if advisoryName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "advisory_id param not found"})
		return
	}

	var exists int
	err := database.Db.Model(&models.AdvisoryMetadata{}).Where("name = ? ", advisoryName).Count(&exists).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
	}
	if exists == 0 {
		LogAndRespNotFound(c, errors.New("System not found"), "System not found")
		return
	}

	query := buildQuery(account, advisoryName)
	path := fmt.Sprintf("/api/patch/v1/advisories/%v/systems", advisoryName)
	query, meta, links, err := ListCommon(query, c, path, SystemsFields, systemsDefaultFilter)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var dbItems []SystemDBLookup

	if err = query.Scan(&dbItems).Error; err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	data := buildAdvisorySystemsData(dbItems)
	var resp = AdvisorySystemsResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildQuery(account, advisoryName string) *gorm.DB {
	query := database.Db.Table("advisory_metadata am").Select(SystemsSelect).
		Joins("join system_advisories sa ON am.id=sa.advisory_id").
		Joins("join system_platform ON sa.system_id=system_platform.id").
		Joins("inner join rh_account ra on system_platform.rh_account_id = ra.id").
		Where("ra.name = ?", account).
		Where("am.name = ?", advisoryName)
	return query
}

func buildAdvisorySystemsData(dbItems []SystemDBLookup) []SystemItem {
	data := make([]SystemItem, len(dbItems))
	for i, model := range dbItems {
		item := SystemItem{
			ID:         model.ID,
			Type:       "system",
			Attributes: model.SystemItemAttributes,
		}
		data[i] = item
	}
	return data
}
