package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

type SystemAdvisoriesResponse struct {
	Data  []AdvisoryItem `json:"data"`  // advisories items
	Links Links          `json:"links"`
	Meta  AdvisoryMeta   `json:"meta"`
}

// @Summary Show me advisories for a system by given inventory id
// @Description Show me advisories for a system by given inventory id
// @ID listSystemAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200 {object} SystemAdvisoriesResponse
// @Router /api/patch/v1/systems/{inventory_id}/advisories [get]
func SystemAdvisoriesHandler(c *gin.Context) {
	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return
	}

	inventoryId := c.Param("inventory_id")
	if inventoryId == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{"inventory_id param not found"})
		return
	}

	query := database.SystemAdvisoriesQueryName(database.Db, inventoryId)

	var total int
	err = query.Count(&total).Error
	if err != nil {
		LogAndRespError(c, err, "error getting items count from db")
		return
	}

	if offset > total {
		c.JSON(http.StatusBadRequest, ErrorResponse{"too big offset"})
		return
	}

	var dbItems []models.SystemAdvisories
	err = query.Limit(limit).Offset(offset).Preload("Advisory").Find(&dbItems).Error
	if gorm.IsRecordNotFoundError(err) {
		LogAndRespNotFound(c, err, "no systems found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	data := buildSystemAdvisoriesData(&dbItems)
	meta := buildAdvisoriesMeta(limit, offset, total)
	links := CreateLinks("/api/patch/v1/systems/$INVENTORY_ID/advisories", offset, limit, total,
		"&data_format=json")
	var resp = SystemAdvisoriesResponse{
		Data: *data,
		Links: links,
		Meta: *meta,
	}
	c.JSON(http.StatusOK, &resp)
	return
}

func buildSystemAdvisoriesData(models *[]models.SystemAdvisories) *[]AdvisoryItem {
	var data []AdvisoryItem
	for _, systemAdvisories := range *models {
		advisory := systemAdvisories.Advisory
		item := AdvisoryItem{Id: advisory.Name, Type: "advisory", Attributes: AdvisoryItemAttributes{
			Description: advisory.Description, Severity: "", PublicDate: advisory.PublicDate, Synopsis: advisory.Synopsis,
			AdvisoryType: advisory.AdvisoryTypeId, ApplicableSystems: 0,
		}}
		data = append(data, item)
	}
	return &data
}
