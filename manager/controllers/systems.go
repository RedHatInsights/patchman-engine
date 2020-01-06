package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

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
	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return
	}

	var total int
	err = database.Db.Model(models.SystemPlatform{}).Count(&total).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	if offset > total {
		c.JSON(http.StatusBadRequest, ErrorResponse{"too big offset"})
		return
	}

	var systems []models.SystemPlatform
	err = database.Db.Limit(limit).Offset(offset).Find(&systems).Error
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
	return
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
			Id:   system.InventoryID,
			Type: "system",
		}
	}
	return &data
}
