package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

type AdvisoriesResponse struct {
	Data  []AdvisoryItem  `json:"data"`
	Links Links           `json:"links"`
	Meta  AdvisoryMeta    `json:"meta"`
}

// @Summary Show me all applicable advisories for all my systems
// @Description Show me all applicable advisories for all my systems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Success 200 {object} AdvisoriesResponse
// @Router /api/patch/v1/advisories [get]
func AdvisoriesListHandler(c *gin.Context) {
	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return
	}

	var total int
	err = database.Db.Model(models.AdvisoryMetadata{}).Count(&total).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	if offset > total {
		c.JSON(http.StatusBadRequest, ErrorResponse{"too big offset"})
		return
	}

	var advisories []models.AdvisoryMetadata
	err = database.Db.Limit(limit).Offset(offset).Find(&advisories).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	data := buildAdvisoriesData(&advisories)
	links := CreateLinks("/api/patch/v1/advisories", offset, limit, total,
		"&data_format=json")
	meta := buildAdvisoriesMeta(limit, offset, total)
	var resp = AdvisoriesResponse{
		Data: *data,
		Links: links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
	return
}

func buildAdvisoriesData(advisories *[]models.AdvisoryMetadata) *[]AdvisoryItem {
	data := make([]AdvisoryItem, len(*advisories))
	for i := 0; i < len(*advisories); i++ {
		advisory := (*advisories)[i]
		data[i] = AdvisoryItem{
			Attributes: AdvisoryItemAttributes{
				// TODO - sync API and DB layout
				Description: advisory.Description,
				Severity: "",
				PublicDate: advisory.PublicDate,
				Synopsis: advisory.Synopsis,
				AdvisoryType: advisory.AdvisoryTypeId,
				// TODO - count using rh-account and advisory_account_data table
				ApplicableSystems: 6 },
			Id: advisory.Name,
			Type: "advisory",
		}
	}
	return &data
}

func buildAdvisoriesMeta(limit, offset, total int) *AdvisoryMeta{
	meta := AdvisoryMeta{
		DataFormat: "json",
		Filter:     nil,
		Limit:      limit,
		Offset:     offset,
		Page:       offset / limit,
		PageSize:   limit,
		Pages:      total / limit,
		TotalItems: total,
	}
	return &meta
}
