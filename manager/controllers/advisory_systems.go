package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

type AdvisorySystemsResponse struct {
	Data  []SystemItem        `json:"data"`
	Links Links               `json:"links"`
	Meta  AdvisorySystemsMeta `json:"meta"`
}

type AdvisorySystemsMeta struct {
	DataFormat string  `json:"data_format"`
	Filter     *string `json:"filter"`
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
	Advisory   string  `json:"advisory"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	Pages      int     `json:"pages"`
	Enabled    bool    `json:"enabled"`
	TotalItems int     `json:"total_items"`
}

// @Summary Show me systems on which the given advisory is applicable
// @Description Show me systems on which the given advisory is applicable
// @ID listAdvisorySystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string   true "Advisory ID"
// @Success 200 {object} AdvisorySystemsResponse
// @Router /api/patch/v1/advisories/{advisory_id}/systems [get]
func AdvisorySystemsListHandler(c *gin.Context) {
	account := c.GetString(middlewares.KEY_ACCOUNT)

	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return
	}

	advisoryName := c.Param("advisory_id")
	if advisoryName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{"advisory_id param not found"})
		return
	}

	query := database.Db.Table("advisory_metadata am").Select("sp.*").
		Joins("join system_advisories sa ON am.id=sa.advisory_id").
		Joins("join system_platform sp ON sa.system_id=sp.id").
		Joins("inner join rh_account ra on sp.rh_account_id = ra.id").
		Where("ra.name = ?", account).
		Where("am.name = ?", advisoryName)

	var total int
	err = query.Count(&total).Error
	if err != nil {
		LogAndRespError(c, err, "error getting items count from db")
		return
	}

	if offset > total {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{"too big offset"})
		return
	}

	var dbItems []models.SystemPlatform
	err = query.Limit(limit).Offset(offset).Scan(&dbItems).Error
	if gorm.IsRecordNotFoundError(err) {
		LogAndRespNotFound(c, err, "no systems found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	data := buildAdvisorySystemsData(&dbItems)
	meta := buildAdvisorySystemsMeta(limit, offset, total, advisoryName)
	links := CreateLinks("/api/patch/v1/advisories/$ADVISORY_ID/systems", offset, limit, total,
		"&data_format=json")
	var resp = AdvisorySystemsResponse{
		Data: *data,
		Links: links,
		Meta: *meta,
	}
	c.JSON(http.StatusOK, &resp)
	return
}

func buildAdvisorySystemsData(dbItems *[]models.SystemPlatform) *[]SystemItem {
	var data []SystemItem
	for _, model := range *dbItems {
		item := SystemItem{Id: model.InventoryID, Type: "system", Attributes: SystemItemAttributes{
			LastUpload: model.LastUpload, Enabled: !model.OptOut, RhsaCount: model.AdvisoryCountCache,
		}}
		data = append(data, item)
	}
	return &data
}

func buildAdvisorySystemsMeta(limit, offset, total int, advisoryName string) *AdvisorySystemsMeta{
	meta := AdvisorySystemsMeta{
			DataFormat: "json",
			Filter: nil,
			Limit: limit,
			Offset: offset,
			Advisory: advisoryName,
			Page: offset / limit,
			PageSize: limit,
			Pages: total / limit,
			Enabled: true,
			TotalItems: total,
		}
	return &meta
}
