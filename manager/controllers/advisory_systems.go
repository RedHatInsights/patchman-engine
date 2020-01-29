package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

type AdvisorySystemsResponse struct {
	Data  []SystemItem `json:"data"`
	Links Links        `json:"links"`
	Meta  ListMeta     `json:"meta"`
}

// @Summary Show me systems on which the given advisory is applicable
// @Description Show me systems on which the given advisory is applicable
// @ID listAdvisorySystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string  true    "Advisory ID"
// @Param    limit          query   int     false   "Limit for paging"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,last_updated,last_evaluation)
// @Success 200 {object} AdvisorySystemsResponse
// @Router /api/patch/v1/advisories/{advisory_id}/systems [get]
func AdvisorySystemsListHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	advisoryName := c.Param("advisory_id")
	if advisoryName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "advisory_id param not found"})
		return
	}

	query := buildQuery(account, advisoryName)
	path := fmt.Sprintf("/api/patch/v1/advisories/%v/systems", advisoryName)
	query, meta, links, err := ListCommon(query, c, SystemsSortFields, path)
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	var dbItems []models.SystemPlatform
	err = query.Scan(&dbItems).Error
	if gorm.IsRecordNotFoundError(err) {
		LogAndRespNotFound(c, err, "no systems found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	data := buildAdvisorySystemsData(&dbItems)
	var resp = AdvisorySystemsResponse{
		Data:  *data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildQuery(account, advisoryName string) *gorm.DB {
	query := database.Db.Table("advisory_metadata am").Select("sp.*").
		Joins("join system_advisories sa ON am.id=sa.advisory_id").
		Joins("join system_platform sp ON sa.system_id=sp.id").
		Joins("inner join rh_account ra on sp.rh_account_id = ra.id").
		Where("ra.name = ?", account).
		Where("am.name = ?", advisoryName)
	return query
}

func buildAdvisorySystemsData(dbItems *[]models.SystemPlatform) *[]SystemItem {
	data := make([]SystemItem, len(*dbItems))
	for i, model := range *dbItems {
		item := SystemItem{
			ID:   model.InventoryID,
			Type: "system",
			Attributes: SystemItemAttributes{
				LastEvaluation: model.LastEvaluation,
				LastUpload:     model.LastUpload,
				RhsaCount:      model.AdvisorySecCountCache,
				RhbaCount:      model.AdvisoryBugCountCache,
				RheaCount:      model.AdvisoryEnhCountCache,
				Enabled:        !model.OptOut,
			}}
		data[i] = item
	}
	return &data
}
