package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"net/http"
)

var SystemsSortFields = []string{"last_updated", "last_evaluation"}

type SystemsResponse struct {
	Data  []SystemItem `json:"data"`
	Links Links        `json:"links"`
	Meta  SystemsMeta  `json:"meta"`
}

type SystemsMeta struct {
	DataFormat string  `json:"data_format"`
	Filter     *string `json:"filter"`
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	Pages      int     `json:"pages"`
	Enabled    bool    `json:"enabled"`
	TotalItems int     `json:"total_items"`
}

// @Summary Show me all my systems
// @Description Show me all my systems
// @ID listSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Success 200 {object} SystemsResponse
// @Router /api/patch/v1/systems [get]
func SystemsListHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return
	}

	query := database.Db.Model(models.SystemPlatform{}).
		Joins("inner join rh_account ra on system_platform.rh_account_id = ra.id").
		Where("ra.name = ?", account)

	query, err = ApplySort(c, query, "", SystemsSortFields...)
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	var total int
	err = query.Count(&total).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	if offset > total {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "too big offset"})
		return
	}

	var systems []models.SystemPlatform
	err = query.Limit(limit).Offset(offset).Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	data := buildData(&systems)
	links := CreateLinks("/api/patch/v1/systems", offset, limit, total,
		"&data_format=json")
	meta := buildMeta(limit, offset, total)
	var resp = SystemsResponse{
		Data:  *data,
		Links: links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildMeta(limit, offset, total int) *SystemsMeta {
	meta := SystemsMeta{
		DataFormat: "json",
		Filter:     nil,
		Limit:      limit,
		Offset:     offset,
		Page:       offset / limit,
		PageSize:   limit,
		Pages:      total / limit,
		Enabled:    true,
		TotalItems: total,
	}
	return &meta
}

func buildData(systems *[]models.SystemPlatform) *[]SystemItem {
	data := make([]SystemItem, len(*systems))
	for i := 0; i < len(*systems); i++ {
		system := (*systems)[i]
		data[i] = SystemItem{
			Attributes: SystemItemAttributes{
				LastEvaluation: system.LastEvaluation,
				LastUpload:     system.LastUpload,
				RhsaCount:      system.AdvisorySecCountCache,
				RheaCount:      system.AdvisoryEnhCountCache,
				RhbaCount:      system.AdvisoryBugCountCache,
				Enabled:        !system.OptOut,
			},
			ID:   system.InventoryID,
			Type: "system",
		}
	}
	return &data
}
